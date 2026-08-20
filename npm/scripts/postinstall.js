"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const https = require("https");
const crypto = require("crypto");
const { execFileSync } = require("child_process");

const REPO = "SocialSight/cli";
const PKG = require("../package.json");
const BIN_DIR = path.join(__dirname, "..", "bin");
const MAX_REDIRECTS = 5;

function mapOs() {
  switch (process.platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`unsupported platform: ${process.platform}`);
  }
}

function mapArch() {
  switch (process.arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`unsupported architecture: ${process.arch}`);
  }
}

// Downloads url to destPath, following redirects (GitHub release assets
// redirect to S3). Returns a promise that resolves once the file is fully
// written.
function download(url, destPath, redirectsLeft) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destPath);
    https
      .get(url, { headers: { "User-Agent": "socialsight-cli-npm-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          file.close();
          fs.unlinkSync(destPath);
          if (redirectsLeft <= 0) {
            reject(new Error(`too many redirects fetching ${url}`));
            return;
          }
          resolve(download(res.headers.location, destPath, redirectsLeft - 1));
          return;
        }
        if (res.statusCode !== 200) {
          file.close();
          fs.unlinkSync(destPath);
          reject(new Error(`GET ${url} failed: HTTP ${res.statusCode}`));
          return;
        }
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
      })
      .on("error", (err) => {
        file.close();
        reject(err);
      });
  });
}

function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filePath));
  return hash.digest("hex");
}

async function main() {
  const version = PKG.version;
  const tag = `v${version}`;
  const platformOs = mapOs();
  const arch = mapArch();
  const ext = platformOs === "windows" ? "zip" : "tar.gz";
  const archiveName = `socialsight_${version}_${platformOs}_${arch}.${ext}`;
  const baseUrl = `https://github.com/${REPO}/releases/download/${tag}`;

  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "socialsight-install-"));
  const archivePath = path.join(tmpDir, archiveName);
  const checksumsPath = path.join(tmpDir, "checksums.txt");

  console.log(`socialsight: downloading ${archiveName}...`);
  await download(`${baseUrl}/${archiveName}`, archivePath, MAX_REDIRECTS);
  await download(`${baseUrl}/checksums.txt`, checksumsPath, MAX_REDIRECTS);

  const checksums = fs.readFileSync(checksumsPath, "utf8");
  const line = checksums.split("\n").find((l) => l.trim().endsWith(archiveName));
  if (!line) {
    throw new Error(`no checksum found for ${archiveName} in checksums.txt`);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual = sha256File(archivePath);
  if (expected !== actual) {
    throw new Error(`checksum mismatch for ${archiveName}: expected ${expected}, got ${actual}`);
  }

  // tar (bsdtar on modern Windows, GNU/BSD tar elsewhere) auto-detects the
  // archive format from content, so this works for both .tar.gz and .zip
  // without any extra npm dependency.
  execFileSync("tar", ["-xf", archivePath, "-C", tmpDir]);

  const extractedName = platformOs === "windows" ? "socialsight.exe" : "socialsight";
  const extractedPath = path.join(tmpDir, extractedName);
  if (!fs.existsSync(extractedPath)) {
    throw new Error(`archive did not contain a ${extractedName} binary`);
  }

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const installedName = platformOs === "windows" ? "socialsight-bin.exe" : "socialsight-bin";
  const installedPath = path.join(BIN_DIR, installedName);
  fs.copyFileSync(extractedPath, installedPath);
  fs.chmodSync(installedPath, 0o755);

  fs.rmSync(tmpDir, { recursive: true, force: true });

  console.log(`socialsight: installed ${tag} for ${platformOs}/${arch}`);
}

main().catch((err) => {
  console.error(`socialsight: install failed: ${err.message}`);
  process.exit(1);
});
