const fs = require("fs");
const path = require("path");
const CleanCSS = require("clean-css");

const root = path.resolve(__dirname, "..");
const files = [
  "fonts.css",
  "theme.css",
  "utility.css",
  "admin.css",
  "seller-admin.css",
  "support-portal.css",
];

for (const file of files) {
  const inputPath = path.join(root, "web", "assets", "css", file);
  const outputPath = path.join(root, "web", "assets", "css", file.replace(/\.css$/, ".min.css"));
  const source = fs.readFileSync(inputPath, "utf8");
  const result = new CleanCSS({ level: 2 }).minify(source);
  if (result.errors.length > 0) {
    throw new Error(`${file}: ${result.errors.join("; ")}`);
  }
  fs.writeFileSync(outputPath, `${result.styles}\n`);
}
