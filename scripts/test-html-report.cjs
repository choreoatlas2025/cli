// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
// Run with: node scripts/test-html-report.cjs /path/to/chrome /path/to/report.html
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { pathToFileURL } = require('node:url');
const { execFileSync } = require('node:child_process');

const [browser, report] = process.argv.slice(2);
if (!browser || !report) throw new Error('browser and Go-generated report paths are required');
const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ce-html-browser-'));
try {
  const checks = `<script>
    document.addEventListener('DOMContentLoaded', () => setTimeout(() => {
      const errors = [];
      if (window.ceInjected) errors.push('external script executed');
      if (document.querySelector('.ce-external, [onerror], [onclick], [onmouseover]')) errors.push('external field created markup');
      if (!document.getElementById('steps-tbody').textContent.includes('<img')) errors.push('external text was lost');
      if (!document.querySelector('.gantt-label').textContent.includes('<img')) errors.push('timeline text was lost');
      if (!document.querySelector('.gantt-bar').title.includes('<img')) errors.push('timeline tooltip was lost');
      const gate = document.getElementById('gate-status').textContent;
      if ((gate.match(/required: ≥0.0%/g) || []).length !== 2) errors.push('zero thresholds rendered incorrectly');
      if (!gate.includes('<img')) errors.push('violation text was lost');
      const result = document.createElement('pre');
      result.id = 'ce-browser-result';
      result.textContent = JSON.stringify({ errors });
      document.body.appendChild(result);
    }, 100));
  </script>`;
  const fixture = path.join(dir, 'checked.html');
  fs.writeFileSync(fixture, fs.readFileSync(report, 'utf8') + checks);
  // Bound the browser's capture wait independently of the process lifetime.
  const dom = execFileSync(browser, ['--headless', '--no-sandbox', '--disable-background-networking', '--disable-extensions', '--no-first-run', '--no-default-browser-check', '--user-data-dir=' + path.join(dir, 'profile'), '--dump-dom', '--timeout=10000', '--virtual-time-budget=1000', pathToFileURL(fixture).href], { encoding: 'utf8', timeout: 30000, maxBuffer: 8 * 1024 * 1024, stdio: ['ignore', 'pipe', 'pipe'] });
  const match = dom.match(/<pre id="ce-browser-result">([^<]*)<\/pre>/);
  if (!match) throw new Error('browser did not complete report checks');
  const result = JSON.parse(match[1]);
  if (result.errors.length) throw new Error(result.errors.join('; '));
  console.log('Browser report checks passed: external fields remain text; no injection executed; zero thresholds preserved.');
} finally {
  fs.rmSync(dir, { recursive: true, force: true });
}
