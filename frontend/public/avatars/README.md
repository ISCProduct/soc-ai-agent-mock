# Avatar Files

This directory should contain 3D avatar GLB files for the AI interview feature.

## Required Files

- `male-avatar.glb` - Male interviewer avatar
- `female-avatar.glb` - Female interviewer avatar

## How to Get Avatar Files

### Option 1: Ready Player Me (Recommended)

1. Go to [Ready Player Me](https://readyplayer.me/)
2. Click "Create Avatar" or "Get Started"
3. Create a male and female avatar using the customization tools
4. For each avatar:
   - Click the download/export button
   - Use this URL format to download with morph targets:
   ```
   https://models.readyplayer.me/[YOUR_AVATAR_ID].glb?morphTargets=Oculus+Visemes&compression=draco
   ```
   - Save as `male-avatar.glb` or `female-avatar.glb` in this directory

### Option 2: Use Free 3D Models

You can use any humanoid GLB model that includes Oculus OVR LipSync viseme morph targets:
- [Mixamo](https://www.mixamo.com/) - Free character models (requires rigging for morph targets)
- [Sketchfab](https://sketchfab.com/) - Search for "avatar" or "character" (filter by glTF/GLB)

### Option 3: Use the Fallback

If no local files are found, the app will:
1. Try to load from Ready Player Me API URLs (if configured)
2. Fall back to a simple SVG avatar illustration

The SVG fallback works perfectly fine for development and testing!

## File Requirements

- **Format**: GLB (binary glTF)
- **Morph Targets**: Must include Oculus OVR LipSync visemes for lipsync
  - `viseme_sil`, `viseme_PP`, `viseme_FF`, `viseme_TH`, `viseme_DD`
  - `viseme_kk`, `viseme_CH`, `viseme_SS`, `viseme_nn`, `viseme_RR`
  - `viseme_aa`, `viseme_E`, `viseme_I`, `viseme_O`, `viseme_U`
- **Size**: Keep under 3MB for fast loading (use DRACO compression)
- **Optimization**: Use DRACO compression for smaller file sizes

## Testing

After adding the files, test by:
1. Starting the dev server: `npm run dev`
2. Going to `/interview`
3. Starting an interview session
4. The 3D avatar should load and show lipsync when AI speaks

If the avatars don't load, check the browser console for error messages.
