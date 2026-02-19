'use client'

import { useEffect, useRef, useState } from 'react'
import * as THREE from 'three'
import { Box } from '@mui/material'
import { loadAvatar, type AvatarGender } from '@/lib/avatar-loader'
import { LipsyncManager } from '@/lib/lipsync-manager'

interface ThreeAvatarProps {
  gender: AvatarGender
  audioStream: MediaStream | null
  level: number
  speaking: boolean
}

export default function ThreeAvatar({
  gender,
  audioStream,
  level,
  speaking,
}: ThreeAvatarProps) {
  const [useFallback, setUseFallback] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const levelRef = useRef(level)
  const speakingRef = useRef(speaking)
  const lipsyncManagerRef = useRef<LipsyncManager | null>(null)
  const avatarMeshRef = useRef<THREE.Object3D | null>(null)
  const morphTargetMeshesRef = useRef<THREE.Mesh[]>([])

  useEffect(() => {
    levelRef.current = level
    speakingRef.current = speaking
  }, [level, speaking])

  // Update lipsync manager when audio stream changes
  useEffect(() => {
    if (lipsyncManagerRef.current && audioStream) {
      lipsyncManagerRef.current.updateAudioStream(audioStream)
    }
  }, [audioStream])

  useEffect(() => {
    if (!containerRef.current || useFallback) return

    const container = containerRef.current
    const width = container.clientWidth
    const height = container.clientHeight

    // Three.js scene setup
    const scene = new THREE.Scene()
    const camera = new THREE.PerspectiveCamera(34, width / height, 0.1, 20)
    camera.position.set(0, 0.8, 2.5)

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2)) // Cap at 2 for performance
    renderer.setSize(width, height)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    renderer.shadowMap.enabled = true
    renderer.shadowMap.type = THREE.PCFSoftShadowMap
    container.appendChild(renderer.domElement)

    // Lighting setup
    scene.add(new THREE.AmbientLight(0xffffff, 1.2))

    const keyLight = new THREE.DirectionalLight(0xffffff, 1.5)
    keyLight.position.set(2, 2, 3)
    keyLight.castShadow = true
    scene.add(keyLight)

    const fillLight = new THREE.DirectionalLight(0xffffff, 0.6)
    fillLight.position.set(-2, 1, -2)
    scene.add(fillLight)

    const rimLight = new THREE.DirectionalLight(0xffffff, 0.4)
    rimLight.position.set(0, 2, -3)
    scene.add(rimLight)

    // Avatar container group
    const avatarGroup = new THREE.Group()
    scene.add(avatarGroup)

    // Background halo
    const halo = new THREE.Mesh(
      new THREE.CircleGeometry(1.5, 40),
      new THREE.MeshBasicMaterial({
        color: 0xf7f4ec,
        transparent: true,
        opacity: 0.3
      })
    )
    halo.position.set(0, 0.5, -0.5)
    scene.add(halo)

    let disposed = false
    let frameId = 0
    let lastMorphUpdateTime = 0
    const morphUpdateInterval = 1000 / 30 // 30fps for morph target updates

    // Animation loop
    const animate = () => {
      if (disposed) return

      const currentTime = performance.now()
      const t = currentTime / 1000

      // Subtle head movement
      const talk = Math.min(1, levelRef.current * 1.2)
      avatarGroup.rotation.y = Math.sin(t * 0.5) * 0.04 + (talk - 0.2) * 0.02
      avatarGroup.rotation.x = Math.sin(t * 0.7) * 0.02
      avatarGroup.position.y = Math.sin(t * 1.0) * 0.03

      // Subtle breathing/pulse effect
      if (speakingRef.current) {
        const pulse = 1 + talk * 0.015
        avatarGroup.scale.set(pulse, pulse, pulse)
      } else {
        const breathe = 1 + Math.sin(t * 0.8) * 0.005
        avatarGroup.scale.set(breathe, breathe, breathe)
      }

      // Update morph targets at 30fps
      if (
        lipsyncManagerRef.current &&
        morphTargetMeshesRef.current.length > 0 &&
        currentTime - lastMorphUpdateTime > morphUpdateInterval
      ) {
        lastMorphUpdateTime = currentTime
        updateMorphTargets()
      }

      renderer.render(scene, camera)
      frameId = requestAnimationFrame(animate)
    }

    // Update morph targets based on lipsync data
    const updateMorphTargets = () => {
      if (!lipsyncManagerRef.current) return

      const visemeWeights = lipsyncManagerRef.current.getCurrentVisemes()

      morphTargetMeshesRef.current.forEach((mesh) => {
        if (!mesh.morphTargetDictionary || !mesh.morphTargetInfluences) return

        // Apply viseme weights to morph targets
        Object.entries(visemeWeights).forEach(([visemeName, weight]) => {
          const index = mesh.morphTargetDictionary![visemeName]
          if (index !== undefined && mesh.morphTargetInfluences) {
            mesh.morphTargetInfluences[index] = weight
          }
        })
      })
    }

    // Find meshes with morph targets in the loaded model
    const findMorphTargetMeshes = (object: THREE.Object3D): THREE.Mesh[] => {
      const meshes: THREE.Mesh[] = []

      object.traverse((child) => {
        if (child instanceof THREE.Mesh && child.morphTargetDictionary) {
          meshes.push(child)
        }
      })

      return meshes
    }

    // Load avatar model
    const loadAvatarModel = async () => {
      try {
        setIsLoading(true)
        console.log(`[ThreeAvatar] Loading ${gender} avatar...`)

        const gltf = await loadAvatar(gender)

        if (disposed) {
          console.log('[ThreeAvatar] Component disposed before avatar loaded')
          return
        }

        const model = gltf.scene

        // Scale and position the avatar
        model.scale.set(1, 1, 1)
        model.position.set(0, -0.8, 0)

        avatarGroup.add(model)
        avatarMeshRef.current = model

        // Find and store meshes with morph targets
        morphTargetMeshesRef.current = findMorphTargetMeshes(model)
        console.log(`[ThreeAvatar] Found ${morphTargetMeshesRef.current.length} meshes with morph targets`)

        // Initialize lipsync manager
        if (audioStream) {
          lipsyncManagerRef.current = new LipsyncManager(audioStream)
          console.log('[ThreeAvatar] LipsyncManager initialized')
        }

        setIsLoading(false)
        animate()

        console.log(`[ThreeAvatar] ${gender} avatar loaded and animated`)
      } catch (error) {
        console.error('[ThreeAvatar] Failed to load avatar:', error)
        if (!disposed) {
          setUseFallback(true)
        }
      }
    }

    loadAvatarModel()

    // Handle window resize
    const onResize = () => {
      const w = container.clientWidth
      const h = container.clientHeight
      camera.aspect = w / h
      camera.updateProjectionMatrix()
      renderer.setSize(w, h)
    }
    window.addEventListener('resize', onResize)

    // Cleanup
    return () => {
      disposed = true
      cancelAnimationFrame(frameId)
      window.removeEventListener('resize', onResize)

      // Dispose lipsync manager
      if (lipsyncManagerRef.current) {
        lipsyncManagerRef.current.dispose()
        lipsyncManagerRef.current = null
      }

      // Dispose Three.js resources
      renderer.dispose()
      if (renderer.domElement.parentElement === container) {
        container.removeChild(renderer.domElement)
      }

      scene.traverse((obj) => {
        if (obj instanceof THREE.Mesh) {
          obj.geometry.dispose()
          const m = obj.material
          if (Array.isArray(m)) {
            m.forEach((mm) => mm.dispose())
          } else {
            m.dispose()
          }
        }
      })

      morphTargetMeshesRef.current = []
      avatarMeshRef.current = null

      console.log('[ThreeAvatar] Cleanup complete')
    }
  }, [gender, useFallback])

  // Fallback to simple SVG avatar
  if (useFallback) {
    return (
      <InterviewerFallbackAvatar
        gender={gender}
        level={level}
        speaking={speaking}
      />
    )
  }

  return (
    <Box
      ref={containerRef}
      sx={{
        width: { xs: 214, md: 330, lg: 380 },
        height: { xs: 214, md: 330, lg: 380 },
        position: 'relative',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      {isLoading && (
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            color: 'text.secondary',
            fontSize: '0.875rem',
          }}
        >
          アバター読込中...
        </Box>
      )}
    </Box>
  )
}

// Fallback SVG avatar component (copied from interview page)
function InterviewerFallbackAvatar({
  gender,
  level,
  speaking,
}: {
  gender: 'male' | 'female'
  level: number
  speaking: boolean
}) {
  const mouthOpen = Math.max(4, Math.min(18, Math.round(4 + level * 20)))
  const hairColor = gender === 'female' ? '#4f3326' : '#2b2b34'
  const suitColor = gender === 'female' ? '#48607f' : '#2f4a66'
  const accentColor = gender === 'female' ? '#e6d8c4' : '#c8d6e5'

  return (
    <Box
      sx={{
        width: { xs: 190, md: 280 },
        height: { xs: 190, md: 280 },
        borderRadius: '50%',
        overflow: 'hidden',
        position: 'relative',
        display: 'grid',
        placeItems: 'center',
        background: 'radial-gradient(circle at 48% 30%, #fefcf7 0%, #f2eadf 38%, #d7c7b0 100%)',
        boxShadow: speaking
          ? '0 0 0 10px rgba(30, 64, 175, 0.15), inset 0 -8px 20px rgba(0,0,0,0.1)'
          : 'inset 0 -8px 20px rgba(0,0,0,0.08)',
        transform: speaking ? 'scale(1.01)' : 'scale(1)',
        transition: 'all 0.16s ease',
      }}
    >
      <svg
        viewBox="0 0 120 140"
        width="100%"
        height="100%"
        style={{ position: 'absolute', top: 0, left: 0 }}
      >
        <defs>
          <radialGradient id="face-grad" cx="50%" cy="40%">
            <stop offset="0%" stopColor="#ffe8d4" />
            <stop offset="100%" stopColor="#f5d5bd" />
          </radialGradient>
        </defs>

        <ellipse cx="60" cy="52" rx="28" ry="32" fill="url(#face-grad)" />

        <ellipse cx="60" cy="25" rx="30" ry="28" fill={hairColor} />
        <rect x="30" y="28" width="60" height="16" fill={hairColor} />

        <circle cx="50" cy="48" r="2.5" fill="#2c2420" />
        <circle cx="70" cy="48" r="2.5" fill="#2c2420" />

        <ellipse cx="48" cy="52" rx="3" ry="2" fill="#ffb8a0" opacity="0.45" />
        <ellipse cx="72" cy="52" rx="3" ry="2" fill="#ffb8a0" opacity="0.45" />

        <path
          d={`M 52 60 Q 60 ${60 + mouthOpen} 68 60`}
          fill="none"
          stroke="#c97a6a"
          strokeWidth="1.8"
          strokeLinecap="round"
        />
        {speaking && mouthOpen > 8 && (
          <ellipse cx="60" cy={64 + mouthOpen / 2} rx="6" ry={mouthOpen / 2} fill="#4a2020" opacity="0.4" />
        )}

        <rect x="32" y="76" width="56" height="50" rx="2" fill={suitColor} />

        <path d="M 60 76 L 52 86 L 60 95 L 68 86 Z" fill={accentColor} />

        <rect x="32" y="76" width="14" height="40" fill={suitColor} opacity="0.85" />
        <rect x="74" y="76" width="14" height="40" fill={suitColor} opacity="0.85" />
      </svg>
    </Box>
  )
}
