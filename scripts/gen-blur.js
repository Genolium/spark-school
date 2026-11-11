const fs = require('fs');
const path = require('path');
// sharp is a frontend dependency; load it from frontend/node_modules
const sharp = require(path.resolve(__dirname, '..', 'frontend', 'node_modules', 'sharp'));

const publicDir = path.resolve(__dirname, '..', 'frontend', 'public');
const outFile = path.resolve(__dirname, '..', 'frontend', 'src', 'lib', 'blurData.ts');

const images = [
  'spatial-campus-hero.webp',
  'spatial-campus-hero-mobile.webp',
  'ilya-author.webp',
  'criteria-essay.webp',
  'criteria-video.webp',
  'criteria-resume.webp',
  'criteria-zoom.webp',
  'stage-01.webp',
  'stage-02.webp',
  'stage-03.webp',
  'stage-04.webp',
  'stage-05.webp'
];

const results = {};

async function main() {
  for (const imgName of images) {
    const imgPath = path.join(publicDir, imgName);
    if (!fs.existsSync(imgPath)) {
      console.warn('Missing:', imgName);
      continue;
    }
    try {
      const buf = await sharp(imgPath).resize(24, 16, { fit: 'cover' }).jpeg().toBuffer();
      const b64 = buf.toString('base64');
      results['/' + imgName] = 'data:image/jpeg;base64,' + b64;
      console.log('Extracted blur thumbnail for', imgName, `(${b64.length} chars)`);
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
}

main();
