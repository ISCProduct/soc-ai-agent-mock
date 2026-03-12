/**
 * LipsyncManager - Manages real-time lipsync using wawa-lipsync library
 * Converts audio stream to viseme data and maps to morph target weights
 */

// Oculus OVR LipSync viseme list (used by Ready Player Me)
// Reference: https://docs.readyplayer.me/ready-player-me/api-reference/avatars/morph-targets/oculus-ovr-libsync
const OCULUS_VISEMES = [
  'viseme_sil',  // silence
  'viseme_PP',   // PP, BB, MB
  'viseme_FF',   // FF, V
  'viseme_TH',   // TH
  'viseme_DD',   // DD, T, N, L
  'viseme_kk',   // K, G, NG, CH, SH, ZH
  'viseme_CH',   // CH, J, SH
  'viseme_SS',   // S, Z
  'viseme_nn',   // N, NG
  'viseme_RR',   // R
  'viseme_aa',   // AA, AO
  'viseme_E',    // EH, AE, UH
  'viseme_I',    // IH
  'viseme_O',    // O
  'viseme_U',    // UW, UH
] as const

// Map from wawa-lipsync phoneme codes to Oculus visemes
const PHONEME_TO_VISEME_MAP: Record<string, string> = {
  // Silence
  'sil': 'viseme_sil',

  // Consonants
  'p': 'viseme_PP',
  'b': 'viseme_PP',
  'm': 'viseme_PP',

  'f': 'viseme_FF',
  'v': 'viseme_FF',

  'th': 'viseme_TH',
  'dh': 'viseme_TH',

  't': 'viseme_DD',
  'd': 'viseme_DD',
  'n': 'viseme_DD',
  'l': 'viseme_DD',

  'k': 'viseme_kk',
  'g': 'viseme_kk',
  'ng': 'viseme_kk',

  'ch': 'viseme_CH',
  'jh': 'viseme_CH',
  'sh': 'viseme_CH',
  'zh': 'viseme_CH',

  's': 'viseme_SS',
  'z': 'viseme_SS',

  'r': 'viseme_RR',

  // Vowels
  'aa': 'viseme_aa',
  'ao': 'viseme_aa',
  'ax': 'viseme_aa',

  'eh': 'viseme_E',
  'ae': 'viseme_E',
  'ah': 'viseme_E',
  'uh': 'viseme_E',

  'ih': 'viseme_I',
  'iy': 'viseme_I',

  'oh': 'viseme_O',
  'ow': 'viseme_O',

  'uw': 'viseme_U',
  'uu': 'viseme_U',
}

export interface VisemeWeights {
  [visemeName: string]: number
}

export class LipsyncManager {
  private audioContext: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private sourceNode: MediaStreamAudioSourceNode | null = null
  private currentViseme: string = 'viseme_sil'
  private targetViseme: string = 'viseme_sil'
  private visemeWeight: number = 0
  private smoothingFactor: number = 0.3
  private animationFrameId: number | null = null
  private dataArray: Uint8Array<ArrayBuffer> | null = null
  private lastUpdateTime: number = 0
  private updateInterval: number = 1000 / 30 // 30fps for morph target updates

  constructor(audioStream: MediaStream | null) {
    if (audioStream) {
      this.initializeAudioAnalysis(audioStream)
    }
  }

  /**
   * Initialize audio analysis from MediaStream
   */
  private initializeAudioAnalysis(audioStream: MediaStream): void {
    try {
      this.audioContext = new AudioContext()
      this.sourceNode = this.audioContext.createMediaStreamSource(audioStream)
      this.analyser = this.audioContext.createAnalyser()

      this.analyser.fftSize = 2048
      this.analyser.smoothingTimeConstant = 0.8
      this.dataArray = new Uint8Array(this.analyser.frequencyBinCount)

      this.sourceNode.connect(this.analyser)

      console.log('[LipsyncManager] Audio analysis initialized')
      this.startAnalysis()
    } catch (error) {
      console.error('[LipsyncManager] Failed to initialize audio analysis:', error)
    }
  }

  /**
   * Start analyzing audio and generating viseme data
   */
  private startAnalysis(): void {
    const analyze = (currentTime: number) => {
      if (!this.analyser || !this.dataArray) {
        return
      }

      // Throttle updates to 30fps
      if (currentTime - this.lastUpdateTime < this.updateInterval) {
        this.animationFrameId = requestAnimationFrame(analyze)
        return
      }
      this.lastUpdateTime = currentTime

      // Get audio data
      this.analyser.getByteFrequencyData(this.dataArray)

      // Calculate RMS (root mean square) for volume
      let sum = 0
      for (let i = 0; i < this.dataArray.length; i++) {
        const normalized = this.dataArray[i] / 255
        sum += normalized * normalized
      }
      const rms = Math.sqrt(sum / this.dataArray.length)

      // Simple viseme selection based on frequency bands
      // This is a simplified approach; wawa-lipsync would provide more accurate phoneme detection
      const viseme = this.selectVisemeFromFrequencies(this.dataArray, rms)

      // Update target viseme
      if (viseme !== this.targetViseme) {
        this.targetViseme = viseme
      }

      // Smooth transition between visemes
      if (this.currentViseme !== this.targetViseme) {
        this.currentViseme = this.targetViseme
        this.visemeWeight = 0
      }

      // Animate weight
      if (rms > 0.01) {
        this.visemeWeight = Math.min(1, this.visemeWeight + this.smoothingFactor)
      } else {
        this.visemeWeight = Math.max(0, this.visemeWeight - this.smoothingFactor * 2)
        if (this.visemeWeight === 0) {
          this.currentViseme = 'viseme_sil'
          this.targetViseme = 'viseme_sil'
        }
      }

      this.animationFrameId = requestAnimationFrame(analyze)
    }

    this.animationFrameId = requestAnimationFrame(analyze)
  }

  /**
   * Select viseme based on frequency analysis
   * This is a simplified heuristic approach
   */
  private selectVisemeFromFrequencies(frequencies: Uint8Array, rms: number): string {
    if (rms < 0.01) {
      return 'viseme_sil'
    }

    // Analyze different frequency bands
    const lowFreq = this.getFrequencyBandEnergy(frequencies, 0, 200)      // 0-200 Hz
    const midFreq = this.getFrequencyBandEnergy(frequencies, 200, 600)    // 200-600 Hz
    const highFreq = this.getFrequencyBandEnergy(frequencies, 600, 2000)  // 600-2000 Hz
    const veryHighFreq = this.getFrequencyBandEnergy(frequencies, 2000, frequencies.length) // 2000+ Hz

    // Simple heuristic mapping (this is a fallback; ideally wawa-lipsync would provide better analysis)
    if (veryHighFreq > highFreq && veryHighFreq > midFreq) {
      return Math.random() > 0.5 ? 'viseme_SS' : 'viseme_FF' // Sibilants
    } else if (highFreq > midFreq * 1.5) {
      return Math.random() > 0.5 ? 'viseme_E' : 'viseme_I' // High vowels
    } else if (midFreq > lowFreq * 1.2) {
      return Math.random() > 0.5 ? 'viseme_aa' : 'viseme_O' // Mid vowels
    } else if (lowFreq > 0.3) {
      return Math.random() > 0.5 ? 'viseme_PP' : 'viseme_DD' // Consonants
    }

    return 'viseme_aa' // Default open mouth for speech
  }

  /**
   * Get energy in a specific frequency band
   */
  private getFrequencyBandEnergy(frequencies: Uint8Array, startIdx: number, endIdx: number): number {
    let sum = 0
    const actualEnd = Math.min(endIdx, frequencies.length)
    for (let i = startIdx; i < actualEnd; i++) {
      sum += frequencies[i] / 255
    }
    return sum / (actualEnd - startIdx)
  }

  /**
   * Get current viseme weights for morph targets
   * Returns an object with viseme names as keys and weights (0-1) as values
   */
  getCurrentVisemes(): VisemeWeights {
    const weights: VisemeWeights = {}

    // Initialize all visemes to 0
    OCULUS_VISEMES.forEach(viseme => {
      weights[viseme] = 0
    })

    // Set current viseme weight
    if (this.currentViseme && this.visemeWeight > 0) {
      weights[this.currentViseme] = this.visemeWeight
    }

    return weights
  }

  /**
   * Update audio stream (for when it changes)
   */
  updateAudioStream(audioStream: MediaStream | null): void {
    this.dispose()
    if (audioStream) {
      this.initializeAudioAnalysis(audioStream)
    }
  }

  /**
   * Clean up resources
   */
  dispose(): void {
    if (this.animationFrameId !== null) {
      cancelAnimationFrame(this.animationFrameId)
      this.animationFrameId = null
    }

    if (this.sourceNode) {
      this.sourceNode.disconnect()
      this.sourceNode = null
    }

    if (this.audioContext) {
      this.audioContext.close()
      this.audioContext = null
    }

    this.analyser = null
    this.dataArray = null

    console.log('[LipsyncManager] Disposed')
  }
}
