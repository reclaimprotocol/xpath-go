// Test individual not() predicates to verify they work
const fs = require('fs');
const { exec } = require('child_process');
const testcases = require('./shared/testcases.json');

const notTests = [
  'Complex nested structure',
  'Navigation breadcrumbs', 
  'Empty elements'
];

console.log('Testing individual not() predicate tests...\n');

function testSingle(testName) {
  const test = testcases.find(t => t.name === testName);
  if (!test) {
    console.log(`❌ Test "${testName}" not found`);
    return;
  }
  
  console.log(`Testing: ${test.name}`);
  console.log(`XPath: ${test.xpath}`);
  console.log(`HTML: ${test.html.substring(0, 100)}...`);
  
  // Write temp files
  fs.writeFileSync('/tmp/test.html', test.html);
  fs.writeFileSync('/tmp/test.xpath', test.xpath);
  
  // Test with Go
  exec(`echo '${test.html}' | go run -C /Users/abdul/Desktop/code/cc_exp/xpath-go . '${test.xpath}'`, (err, stdout, stderr) => {
    if (err) {
      console.log(`❌ Go error: ${stderr}`);
    } else {
      const goResults = JSON.parse(stdout.trim());
      console.log(`Go results: ${goResults.length} matches`);
      goResults.forEach((r, i) => console.log(`  ${i+1}: "${r.textContent}"`));
    }
    console.log('');
  });
}

notTests.forEach(testName => testSingle(testName));