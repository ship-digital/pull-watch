const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

// --- Configuration ---
const GORELEASER_DIST = path.resolve(process.cwd(), "dist");
const TEMPLATE_PACKAGE_JSON = path.resolve(process.cwd(), "package.json");
const NPM_BUILD_DIR = path.resolve(GORELEASER_DIST, "npm-postinstall-build"); // Dir to store final .tgz file
const TEMP_WORK_DIR = path.resolve(
  GORELEASER_DIST,
  "npm-postinstall-temp-work"
); // Dir for temporary package building
const GITHUB_REF = process.env.GITHUB_REF; // e.g., refs/tags/v1.2.3
const RELEASE_TAG = process.env.RELEASE_TAG; // e.g., v1.2.3 - directly from workflow
const NPM_TOKEN = process.env.NODE_AUTH_TOKEN;
const NPM_PROVENANCE = process.env.NPM_CONFIG_PROVENANCE === "true";
// --- End Configuration ---

// --- Helper Functions ---

function log(message) {
  console.log(`[publish-npm] ${message}`);
}

function error(message) {
  console.error(`[publish-npm] ERROR: ${message}`);
  process.exit(1);
}

function getVersionFromTag(ref) {
  // First try to use RELEASE_TAG if available
  if (RELEASE_TAG) {
    if (RELEASE_TAG.startsWith("v")) {
      return RELEASE_TAG.substring(1); // Remove 'v' prefix
    }
    return RELEASE_TAG; // Use as-is if no 'v' prefix
  }

  // Fall back to GITHUB_REF
  if (!ref || !ref.startsWith("refs/tags/v")) {
    error(
      `Invalid or missing GITHUB_REF tag: ${ref}. Must start with 'refs/tags/v' or set RELEASE_TAG env variable.`
    );
  }
  return ref.substring("refs/tags/v".length);
}

function runCommand(command, cwd) {
  log(`Running: ${command} in ${cwd || "current dir"}`);
  try {
    const output = execSync(command, { stdio: "inherit", cwd }); // Show output
    return output;
  } catch (e) {
    error(`Command failed: ${command}\n${e}`);
  }
}

// --- Main Script ---

function main() {
  log("Starting simplified npm publish script (postinstall method)...");

  if (!NPM_TOKEN) {
    error("NODE_AUTH_TOKEN environment variable is not set.");
  }

  // 1. Read Inputs
  if (!fs.existsSync(TEMPLATE_PACKAGE_JSON)) {
    error(`Template package.json file not found: ${TEMPLATE_PACKAGE_JSON}`);
  }

  const templatePackageJson = JSON.parse(
    fs.readFileSync(TEMPLATE_PACKAGE_JSON, "utf8")
  );
  const version = getVersionFromTag(GITHUB_REF);
  const mainPackageName = templatePackageJson.name;

  log(`Publishing version: ${version}`);
  log(`Package name: ${mainPackageName}`);

  // 2. Prepare Directories
  fs.rmSync(NPM_BUILD_DIR, { recursive: true, force: true });
  fs.rmSync(TEMP_WORK_DIR, { recursive: true, force: true });
  fs.mkdirSync(NPM_BUILD_DIR, { recursive: true });
  fs.mkdirSync(TEMP_WORK_DIR, { recursive: true });

  // 3. Build Main Package
  log("Building the main package...");
  const mainPackageTempDir = path.join(TEMP_WORK_DIR, "main-package");
  fs.mkdirSync(mainPackageTempDir, { recursive: true });

  // Prepare main package.json
  const finalMainPackageJson = { ...templatePackageJson }; // Copy template
  finalMainPackageJson.version = version; // Set correct version
  // Remove devDependencies before packing
  delete finalMainPackageJson.devDependencies;
  // Ensure no optionalDependencies are lingering (should be removed from template already)
  delete finalMainPackageJson.optionalDependencies;

  fs.writeFileSync(
    path.join(mainPackageTempDir, "package.json"),
    JSON.stringify(finalMainPackageJson, null, 2)
  );

  // Copy essential files defined in template's 'files'
  if (templatePackageJson.files) {
    templatePackageJson.files.forEach((f) => {
      const sourcePath = path.resolve(process.cwd(), f);
      const destPath = path.join(mainPackageTempDir, path.basename(f)); // Place in root of package
      if (fs.existsSync(sourcePath)) {
        const stats = fs.statSync(sourcePath);
        if (stats.isFile()) {
          fs.copyFileSync(sourcePath, destPath);
          log(`Copied file to main package: ${f}`);
        } else {
          log(`Skipping non-file entry from 'files': ${f}`); // Don't copy 'bin' dir etc.
        }
      } else {
        log(
          `WARN: File specified in template 'files' not found: ${sourcePath}`
        );
      }
    });
  } else {
    error(
      "Template package.json 'files' array is missing or empty. Cannot determine files to include."
    );
  }

  // 4. Pack the main package
  runCommand("npm pack", mainPackageTempDir);
  // npm pack converts `@scope/pkg` to `scope-pkg-version.tgz`
  const mainTgzName = `${mainPackageName
    .replace("@", "")
    .replace("/", "-")}-${version}.tgz`;
  const mainPackedTgzPath = path.join(mainPackageTempDir, mainTgzName);
  const mainFinalTgzPath = path.join(NPM_BUILD_DIR, mainTgzName);

  if (!fs.existsSync(mainPackedTgzPath)) {
    error(`Packed main tgz not found after 'npm pack': ${mainPackedTgzPath}`);
  }
  fs.renameSync(mainPackedTgzPath, mainFinalTgzPath); // Move to final build dir
  log(`Packed main package: ${mainFinalTgzPath}`);

  // 5. Publish the main package
  log(`Publishing package to npm...`);
  const publishCmdBase = `npm publish --access public`;
  const publishCmdProvenance = NPM_PROVENANCE ? "--provenance" : "";

  runCommand(`${publishCmdBase} "${mainFinalTgzPath}" ${publishCmdProvenance}`);
  log(`Published: ${path.basename(mainFinalTgzPath)}`);

  // 6. Cleanup
  log("Cleaning up temporary directories...");
  fs.rmSync(TEMP_WORK_DIR, { recursive: true, force: true });

  log("Simplified npm publish script finished successfully!");
}

// --- Execute ---
main();
