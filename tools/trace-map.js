#!/usr/bin/env node
/**
 * Minimal Trace-Spec Alignment Matcher
 *
 * Compares trace steps against flowspec expectations
 * Outputs: match rate, missing steps, extra steps
 *
 * Usage:
 *   node trace-map.js --trace <path> --flow <path>
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

// Parse command line arguments
const args = process.argv.slice(2);
const getArg = (flag) => {
  const index = args.indexOf(flag);
  return index !== -1 ? args[index + 1] : null;
};

const tracePath = getArg('--trace');
const flowPath = getArg('--flow');

if (!tracePath || !flowPath) {
  console.error('Usage: node trace-map.js --trace <path> --flow <path>');
  process.exit(1);
}

// Read and parse files
const traceData = JSON.parse(fs.readFileSync(path.resolve(tracePath), 'utf8'));
const flowData = yaml.load(fs.readFileSync(path.resolve(flowPath), 'utf8'));

// Extract expected steps from flowspec
const expectedSteps = flowData.flow.map(step => {
  const [service, operation] = step.call.split('.');
  return `${service}.${operation}`;
});

// Extract actual steps from trace
const actualSteps = traceData.steps.map(step =>
  `${step.service}.${step.operation}`
);

// Calculate alignment
const expectedSet = new Set(expectedSteps);
const actualSet = new Set(actualSteps);

let hits = 0;
expectedSteps.forEach(step => {
  if (actualSet.has(step)) hits++;
});

const missing = expectedSteps.filter(step => !actualSet.has(step));
const extra = actualSteps.filter(step => !expectedSet.has(step));
const matchRate = expectedSteps.length > 0 ? hits / expectedSteps.length : 1;

// Generate report
const report = {
  matchRate: Number(matchRate.toFixed(2)),
  threshold: 0.90,
  passed: matchRate >= 0.90,
  summary: {
    expected: expectedSteps.length,
    actual: actualSteps.length,
    matched: hits
  },
  missing: missing,
  extra: extra,
  expectedSteps: expectedSteps,
  actualSteps: actualSteps
};

// Output JSON report
console.log(JSON.stringify(report, null, 2));

// Exit with appropriate code
process.exit(report.passed ? 0 : 1);