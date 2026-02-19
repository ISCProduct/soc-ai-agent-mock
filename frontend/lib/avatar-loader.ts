import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { DRACOLoader } from 'three/examples/jsm/loaders/DRACOLoader.js'
import type { GLTF } from 'three/examples/jsm/loaders/GLTFLoader.js'

const avatarCache = new Map<string, GLTF>()
const loadingPromises = new Map<string, Promise<GLTF>>()

const AVATAR_PATHS = {
  male: '/avatars/male-avatar.glb',
  female: '/avatars/female-avatar.glb',
} as const

const READY_PLAYER_ME_FALLBACK = {
  // Using Ready Player Me demo avatars - replace with your own custom avatars
  male: 'https://models.readyplayer.me/6746bc1f14c5f70f03c7c45a.glb?morphTargets=Oculus+Visemes&compression=draco',
  female: 'https://models.readyplayer.me/6746bdc914c5f70f03c7c45b.glb?morphTargets=Oculus+Visemes&compression=draco',
} as const

export type AvatarGender = 'male' | 'female'

/**
 * Load a 3D avatar model with caching and fallback support
 * @param gender - The gender of the avatar to load ('male' or 'female')
 * @returns Promise that resolves to the loaded GLTF model
 * @throws Error if loading fails after trying both local and remote sources
 */
export async function loadAvatar(gender: AvatarGender): Promise<GLTF> {
  const cacheKey = `avatar-${gender}`

  // Return cached avatar if available
  if (avatarCache.has(cacheKey)) {
    return avatarCache.get(cacheKey)!
  }

  // Return existing loading promise if in progress
  if (loadingPromises.has(cacheKey)) {
    return loadingPromises.get(cacheKey)!
  }

  // Create new loading promise
  const loadingPromise = loadAvatarInternal(gender)
  loadingPromises.set(cacheKey, loadingPromise)

  try {
    const gltf = await loadingPromise
    avatarCache.set(cacheKey, gltf)
    return gltf
  } finally {
    loadingPromises.delete(cacheKey)
  }
}

/**
 * Internal function to load avatar with fallback logic
 */
async function loadAvatarInternal(gender: AvatarGender): Promise<GLTF> {
  const loader = createGLTFLoader()
  const localPath = AVATAR_PATHS[gender]

  // Try loading from local file first
  try {
    console.log(`[AvatarLoader] Loading local avatar: ${localPath}`)
    const gltf = await loadWithTimeout(loader, localPath, 10000)
    console.log(`[AvatarLoader] Successfully loaded local avatar`)
    validateAvatarMorphTargets(gltf)
    return gltf
  } catch (localError) {
    console.warn(`[AvatarLoader] Local avatar not found. Using fallback SVG avatar.`)
    console.info(
      `[AvatarLoader] To use 3D avatars, add GLB files to frontend/public/avatars/\n` +
      `See frontend/public/avatars/README.md for instructions.`
    )

    // Skip Ready Player Me fallback for now (URLs need to be configured)
    // If you want to use Ready Player Me, create your own avatars and update READY_PLAYER_ME_FALLBACK URLs

    throw new Error(
      `Avatar file not found: ${localPath}. ` +
      `Add ${gender}-avatar.glb to frontend/public/avatars/ or see README.md for setup instructions.`
    )
  }
}

/**
 * Create a GLTF loader with DRACO compression support
 */
function createGLTFLoader(): GLTFLoader {
  const loader = new GLTFLoader()

  // Set up DRACO loader for compressed models
  const dracoLoader = new DRACOLoader()
  dracoLoader.setDecoderPath('https://www.gstatic.com/draco/versioned/decoders/1.5.6/')
  dracoLoader.preload()
  loader.setDRACOLoader(dracoLoader)

  return loader
}

/**
 * Load a GLTF model with timeout
 */
function loadWithTimeout(
  loader: GLTFLoader,
  url: string,
  timeoutMs: number
): Promise<GLTF> {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      reject(new Error(`Loading timeout after ${timeoutMs}ms`))
    }, timeoutMs)

    loader.load(
      url,
      (gltf) => {
        clearTimeout(timeoutId)
        resolve(gltf)
      },
      undefined,
      (error) => {
        clearTimeout(timeoutId)
        reject(error)
      }
    )
  })
}

/**
 * Validate that the loaded avatar has required morph targets for lipsync
 */
function validateAvatarMorphTargets(gltf: GLTF): void {
  let hasMorphTargets = false

  gltf.scene.traverse((child: any) => {
    if (child.isMesh && child.morphTargetDictionary) {
      const morphTargets = Object.keys(child.morphTargetDictionary)
      if (morphTargets.length > 0) {
        hasMorphTargets = true
        console.log(`[AvatarLoader] Found ${morphTargets.length} morph targets:`, morphTargets)
      }
    }
  })

  if (!hasMorphTargets) {
    console.warn('[AvatarLoader] Warning: Avatar does not have morph targets. Lipsync will not work.')
  }
}

/**
 * Clear the avatar cache
 */
export function clearAvatarCache(): void {
  avatarCache.clear()
  console.log('[AvatarLoader] Avatar cache cleared')
}

/**
 * Preload avatars for both genders
 */
export async function preloadAvatars(): Promise<void> {
  console.log('[AvatarLoader] Preloading avatars...')
  await Promise.all([
    loadAvatar('male').catch(e => console.error('Failed to preload male avatar:', e)),
    loadAvatar('female').catch(e => console.error('Failed to preload female avatar:', e)),
  ])
  console.log('[AvatarLoader] Avatar preloading complete')
}
