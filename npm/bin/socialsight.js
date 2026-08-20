#!/usr/bin/env node
"use strict";

const path = require("path");
const { spawnSync } = require("child_process");

const binName = process.platform === "win32" ? "socialsight-bin.exe" : "socialsight-bin";
const binPath = path.join(__dirname, binName);

const result = spawnSync(binPath, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  if (result.error.code === "ENOENT") {
    console.error(
      "socialsight: the platform binary is missing -- try reinstalling this package " +
        "(npm install @socialsight/cli --force)"
    );
  } else {
    console.error(`socialsight: failed to launch: ${result.error.message}`);
  }
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
