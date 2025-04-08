// postinstall.js
const fs = require("fs");
const path = require("path");
const os = require("os");
const tar = require("tar");
const AdmZip = require("adm-zip");
const axios = require("axios");

const GITHUB_REPO_OWNER = "ship-digital";
const GITHUB_REPO_NAME = "pull-watch";
const PKG_BIN_DIR_NAME = "bin"; // Matches 'files' and 'bin' in package.json

// --- Helper Functions ---

function log(message) {
  console.log(`[pull-watch-postinstall] ${message}`);
}

function error(message) {
  console.error(`[pull-watch-postinstall] ERROR: ${message}`);
  process.exit(1);
}

function getNodeOs() {
  const platform = os.platform();
  switch (platform) {
    case "linux":
      return "Linux";
    case "darwin":
      return "Darwin";
    case "win32":
      return "Windows";
    default:
      return null; // Or map to Go equivalents if needed
  }
}

function getNodeArch() {
  const arch = os.arch();
  switch (arch) {
    case "x64":
      return "x86_64";
    case "arm64":
      return "arm64";
    // Add other mappings if needed (ia32 -> i386?)
    default:
      return null;
  }
}

function getGoReleaserAssetName(version, nodeOs, nodeArch) {
  // Construct asset name based on goreleaser's default archive.name_template
  // Example: pull-watch_Linux_x86_64.tar.gz
  // Example: pull-watch_Windows_x86_64.zip
  // Adjust this logic EXACTLY match your .goreleaser.yaml `archives.name_template` output!
  const baseName = GITHUB_REPO_NAME; // Assumes project name matches repo name
  const ext = nodeOs === "Windows" ? "zip" : "tar.gz";
  return `${baseName}_${nodeOs}_${nodeArch}.${ext}`;
}

function getBinaryName(targetOs) {
  // Get the binary name from the parent package.json's 'bin' field
  try {
    const parentPackageJsonPath = path.resolve(__dirname, "package.json");
    const parentPackageJson = JSON.parse(
      fs.readFileSync(parentPackageJsonPath, "utf8")
    );
    const binKey = Object.keys(parentPackageJson.bin)[0];
    return targetOs === "Windows" ? `${binKey}.exe` : binKey;
  } catch (e) {
    error(
      `Failed to read or parse package.json to determine binary name: ${e}`
    );
    return targetOs === "Windows" ? "pull-watch.exe" : "pull-watch"; // Fallback
  }
}

// --- Main Postinstall Logic ---

async function main() {
  log("Starting postinstall script...");

  // 1. Determine Platform and Version
  const nodeOs = getNodeOs();
  const nodeArch = getNodeArch();
  let version;
  try {
    version = require("./package.json").version; // Read version from own package.json
  } catch (e) {
    error(`Failed to read version from package.json: ${e}`);
  }

  if (!nodeOs || !nodeArch) {
    error(
      `Unsupported platform or architecture: ${os.platform()}/${os.arch()}. Cannot download binary.`
    );
  }
  log(`Detected platform: ${nodeOs} ${nodeArch}`);
  log(`Required version: ${version}`);

  // 2. Construct Download URL
  const assetName = getGoReleaserAssetName(version, nodeOs, nodeArch);
  const downloadUrl = `https://github.com/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}/releases/download/v${version}/${assetName}`;
  log(`Attempting to download binary package from: ${downloadUrl}`);

  // 3. Download the Archive
  let response;
  try {
    response = await axios({
      url: downloadUrl,
      method: "GET",
      responseType: "arraybuffer", // Important for binary data
      headers: {
        Accept: "application/octet-stream",
      },
    });
    log(`Successfully downloaded ${assetName}`);
  } catch (err) {
    if (err.response) {
      error(
        `Failed to download binary. Status: ${err.response.status}. URL: ${downloadUrl}. Error: ${err.message}`
      );
    } else {
      error(
        `Failed to download binary. URL: ${downloadUrl}. Error: ${err.message}`
      );
    }
  }

  // 4. Prepare Extraction Target
  const targetDir = path.resolve(__dirname); // Install into the package's root directory
  const binDir = path.join(targetDir, PKG_BIN_DIR_NAME);
  const binaryName = getBinaryName(nodeOs);
  const binaryPath = path.join(binDir, binaryName);

  log(`Ensuring target directory exists: ${binDir}`);
  fs.mkdirSync(binDir, { recursive: true });

  // 5. Extract the Binary
  try {
    log(`Extracting ${binaryName} from downloaded archive...`);
    if (assetName.endsWith(".tar.gz")) {
      // Use tar's stream capability
      await new Promise((resolve, reject) => {
        const extractor = tar.x({
          cwd: binDir,
          // Only extract the binary file, assumes it's at the root of the tarball
          filter: (p) => p === binaryName,
          strip: 0,
          onentry: (entry) => {
            entry.path = binaryName;
          }, // Ensure correct final name
        });
        extractor.on("error", reject);
        extractor.on("finish", resolve);
        // Create a readable stream from the buffer and pipe it
        const Readable = require("stream").Readable;
        const bufferStream = new Readable();
        bufferStream.push(response.data);
        bufferStream.push(null); // Signal end of stream
        bufferStream.pipe(extractor);
      });
    } else if (assetName.endsWith(".zip")) {
      const zip = new AdmZip(response.data);
      // Assumes binary is at the root of the zip
      zip.extractEntryTo(binaryName, binDir, false, true);
    } else {
      error(`Unsupported archive format: ${assetName}`);
    }

    if (!fs.existsSync(binaryPath)) {
      error(`Binary not found after extraction: ${binaryPath}`);
    }

    log(`Successfully extracted binary to: ${binaryPath}`);

    // 6. Make Binary Executable (non-Windows)
    if (nodeOs !== "Windows") {
      log(`Setting executable permission for ${binaryPath}`);
      fs.chmodSync(binaryPath, 0o755);
    }
  } catch (e) {
    error(`Failed to extract binary from archive: ${e}`);
  }

  log("Postinstall script completed successfully.");
}

// --- Run ---
main();
