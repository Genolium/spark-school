const fs = require('fs');
const path = require('path');
const cp = require('child_process');

const publicDir = path.resolve(__dirname, '..', 'frontend', 'public');
const outFile = path.resolve(__dirname, '..', 'frontend', 'src', 'lib', 'blurData.ts');

const images = [
  'spatial-campus-hero.jpg',
  'spatial-campus-hero-mobile.jpg',
  'ilya-author.jpg',
  'criteria-essay.jpg',
  'criteria-video.jpg',
  'criteria-resume.jpg',
  'criteria-zoom.jpg',
  'stage-01.jpg',
  'stage-02.jpg',
  'stage-03.jpg',
  'stage-04.jpg',
  'stage-05.jpg'
];

const results = {};

for (const imgName of images) {
  const imgPath = path.join(publicDir, imgName);
  if (!fs.existsSync(imgPath)) {
    console.warn('Missing:', imgName);
    continue;
  }

  // Use powershell to extract thumbnail base64
  const psCmd = `powershell -NoProfile -Command "Add-Type -AssemblyName System.Drawing; $img = [System.Drawing.Image]::FromFile('${imgPath.replace(/\\/g, '/')}'); $thumb = $img.GetThumbnailImage(24, 16, $null, [IntPtr]::Zero); $ms = [System.IO.MemoryStream]::new(); $thumb.Save($ms, [System.Drawing.Imaging.ImageFormat]::Jpeg); $b64 = [Convert]::ToBase64String($ms.ToArray()); $img.Dispose(); $thumb.Dispose(); $ms.Dispose(); [Console]::Write($b64)"`;
  try {
    const b64 = cp.execSync(psCmd, { encoding: 'utf8' }).trim();
    if (b64) {
      results['/' + imgName] = 'data:image/jpeg;base64,' + b64;
      console.log('✓ Extracted blur thumbnail for', imgName, `(${b64.length} chars)`);
    }
  } catch (err) {
    console.error('Error on', imgName, err.message);
  }
}

const fileContent = `// Auto-generated ultra-lightweight progressive blur-up placeholders (24x16 JPEG base64)
// Standard Core Web Vitals technique for instantaneous visual feedback without layout shift

export const blurDataMap: Record<string, string> = ${JSON.stringify(results, null, 2)};

const DEFAULT_BLUR =
  "data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Crect width='100%25' height='100%25' fill='%2312161f'/%3E%3C/svg%3E";

export function getBlurDataURL(src: string | undefined): string {
  if (!src) return DEFAULT_BLUR;
  const cleanSrc = src.split('?')[0];
  return blurDataMap[cleanSrc] || DEFAULT_BLUR;
}
`;

fs.writeFileSync(outFile, fileContent, 'utf8');
console.log('Successfully wrote blurData.ts to:', outFile);
