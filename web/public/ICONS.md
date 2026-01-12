# Icon Files for Cartographus PWA

This directory contains icon files for the Progressive Web App (PWA) manifest and various browser/platform compatibility.

## Icon Files

### Source
- **icon.svg** - Source SVG icon (512x512 viewBox)
  - Primary icon with map pin design
  - Used as fallback for modern browsers
  - Infinitely scalable, smallest file size

### Generated PNG Files (Required)
The following PNG files must be generated from the SVG source:

- **icon-192.png** (192x192) - Required for Android/Chrome PWA
- **icon-512.png** (512x512) - Required for Android/Chrome PWA

### Optional Additional Sizes
- **icon-16.png** (16x16) - Favicon
- **icon-32.png** (32x32) - Favicon
- **icon-180.png** (180x180) - Apple touch icon (iPhone)
- **icon-167.png** (167x167) - Apple touch icon (iPad)

## Generating PNG Icons

### Automated Generation

Run the provided script to automatically generate all required PNG icons:

```bash
./scripts/generate-icons.sh
```

This script will:
1. Check for required tools (rsvg-convert or ImageMagick)
2. Generate PNG icons at all required sizes
3. Output files to `web/public/`

### Manual Generation

#### Using rsvg-convert (Recommended for best quality)

```bash
# Install librsvg2-bin (Debian/Ubuntu)
sudo apt-get install librsvg2-bin

# Generate icons
cd web/public
rsvg-convert -w 192 -h 192 icon.svg -o icon-192.png
rsvg-convert -w 512 -h 512 icon.svg -o icon-512.png
```

#### Using ImageMagick

```bash
# Install ImageMagick
sudo apt-get install imagemagick

# Generate icons
cd web/public
convert -background none -resize 192x192 icon.svg icon-192.png
convert -background none -resize 512x512 icon.svg icon-512.png
```

#### Using Online Tools

1. Upload `icon.svg` to a PNG converter service
2. Export at 192x192 and 512x512 resolutions
3. Save as `icon-192.png` and `icon-512.png`
4. Place in `web/public/` directory

### Verification

After generating icons, verify:

1. **File sizes** - PNG files should exist:
   ```bash
   ls -lh web/public/icon-*.png
   ```

2. **Dimensions** - Check actual pixel dimensions:
   ```bash
   file web/public/icon-*.png
   ```

3. **Visual quality** - Open each PNG and verify clarity
4. **PWA manifest** - Verify manifest.json references are correct
5. **Browser test** - Test PWA installation in Chrome/Edge

## Docker Build Integration

The PNG icons can be generated during Docker build or provided as pre-built assets:

### Option 1: Pre-generate icons (Recommended)
Generate icons locally before building Docker image:
```bash
./scripts/generate-icons.sh
git add web/public/icon-*.png
git commit -m "Add generated PNG icons"
docker build -t cartographus .
```

### Option 2: Generate during build
Add to Dockerfile (requires installing rsvg-convert or ImageMagick in build stage):
```dockerfile
RUN apt-get update && apt-get install -y librsvg2-bin && \
    cd /build/web/public && \
    rsvg-convert -w 192 -h 192 icon.svg -o icon-192.png && \
    rsvg-convert -w 512 -h 512 icon.svg -o icon-512.png
```

## Icon Design Guidelines

If modifying the icon design (icon.svg):

1. **Viewbox**: Use `viewBox="0 0 512 512"` for consistent scaling
2. **Background**: Dark background (#1a1a2e) matching app theme
3. **Content**: High contrast elements for visibility
4. **Safe area**: Keep important elements within 90% of canvas
5. **Maskable**: Design should work as maskable icon (no text near edges)
6. **File size**: Keep SVG under 10KB for fast loading

## Testing Icons

### PWA Installation Test
1. Open app in Chrome/Edge
2. Click "Install" or "Add to Home Screen"
3. Verify icon appears correctly on home screen
4. Verify icon appears in app switcher

### Manifest Validation
Use Chrome DevTools:
1. Open DevTools (F12)
2. Go to Application tab
3. Click "Manifest" in left sidebar
4. Verify all icons load without errors

### Multiple Device Test
Test on:
- Android phone (Chrome)
- iPhone (Safari)
- iPad (Safari)
- Desktop (Chrome, Firefox, Edge)

## Troubleshooting

**Icons not showing in PWA:**
- Clear browser cache
- Uninstall and reinstall PWA
- Verify PNG files exist and are valid
- Check manifest.json syntax

**Icons appear blurry:**
- Ensure PNG files are exact required dimensions
- Don't upscale smaller images
- Regenerate from SVG source

**Build fails with icon errors:**
- Ensure PNG files exist before building
- Run `./scripts/generate-icons.sh` first
- Commit PNG files to repository

## File Checklist

Before deployment, verify these files exist:

- [ ] web/public/icon.svg (source)
- [ ] web/public/icon-192.png (generated)
- [ ] web/public/icon-512.png (generated)
- [ ] web/public/manifest.json (references icons)
- [ ] web/public/index.html (references apple-touch-icon)

## References

- [PWA Icon Guidelines](https://web.dev/add-manifest/)
- [Maskable Icons](https://web.dev/maskable-icon/)
- [Apple Touch Icons](https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariWebContent/ConfiguringWebApplications/ConfiguringWebApplications.html)
