#!/usr/bin/env node
const path = require("path");
const { spawnSync } = require("child_process");
const fs = require("fs");

// Determine the expected path to the actual binary relative to this script
const binaryDir = path.join(__dirname, "bin");
const binaryName =
  process.platform === "win32" ? "pull-watch.exe" : "pull-watch";
const binaryPath = path.join(binaryDir, binaryName);

// Check if the binary exists (postinstall should have run)
if (!fs.existsSync(binaryPath)) {
  console.error(`[pull-watch] Error: Binary not found at ${binaryPath}`);
  console.error(
    "[pull-watch] Postinstall script might have failed. Try reinstalling the package."
  );
  // Attempt to run postinstall manually? Risky, might not have perms or deps.
  // Or suggest a specific reinstall command.
  process.exit(1);
}

// Spawn the actual binary, passing all arguments (slice(2) removes 'node' and 'cli.js')
// Use spawnSync for simplicity as CLI tools are often synchronous
const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit", // Pass stdin, stdout, stderr directly through
});

// Exit with the same code as the binary process if it returns one
if (result.error) {
  console.error(
    `[pull-watch] Error spawning binary ${binaryPath}: ${result.error}`
  );
  process.exit(1); // Exit with failure
}

process.exit(result.status === null ? 0 : result.status); // Exit with binary's exit code
