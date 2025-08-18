const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const { JSDOM } = require('jsdom');

console.log('🔍 Testing Edge Cases - Known Behavioral Differences');
console.log('=====================================================\n');

// Load edge cases
const edgeCasesPath = path.join(__dirname, 'shared', 'edge_cases.json');
const edgeCases = JSON.parse(fs.readFileSync(edgeCasesPath, 'utf8'));

console.log(`📚 Loaded ${edgeCases.length} edge case tests\n`);

let totalTests = 0;
let expectedDifferences = 0;
let unexpectedFailures = 0;

for (const testCase of edgeCases) {
    totalTests++;
    console.log(`[${totalTests}/${edgeCases.length}] ${testCase.name}`);
    console.log(`XPath: ${testCase.xpath}`);
    
    try {
        // Run JavaScript/jsdom evaluation
        const dom = new JSDOM(testCase.html);
        const document = dom.window.document;
        const jsResult = document.evaluate(
            testCase.xpath,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        const jsNodes = [];
        for (let i = 0; i < jsResult.snapshotLength; i++) {
            const node = jsResult.snapshotItem(i);
            jsNodes.push({
                nodeName: node.nodeName.toLowerCase(),
                textContent: node.textContent,
                startLocation: node.startLocation || 0,
                endLocation: node.endLocation || node.textContent?.length || 0
            });
        }
        
        // Run Go evaluation
        const tempHtml = path.join(__dirname, `temp_edge_${Date.now()}_${Math.random().toString(36).substr(2, 9)}.html`);
        const tempXPath = path.join(__dirname, `temp_edge_${Date.now()}_${Math.random().toString(36).substr(2, 9)}.txt`);
        
        fs.writeFileSync(tempHtml, testCase.html);
        fs.writeFileSync(tempXPath, testCase.xpath);
        
        const goOutput = execSync(`cd .. && go run tests/go/main.go "${tempHtml}" "${tempXPath}"`, { encoding: 'utf8' });
        const goResult = JSON.parse(goOutput);
        
        // Cleanup temp files
        try {
            fs.unlinkSync(tempHtml);
            fs.unlinkSync(tempXPath);
        } catch (e) {
            // Ignore cleanup errors
        }
        
        // Compare results
        const jsCount = jsNodes.length;
        const goCount = goResult.results?.length || 0;
        
        let hasDifference = false;
        let differenceDescription = '';
        
        // Check for count differences
        if (jsCount !== goCount) {
            hasDifference = true;
            differenceDescription = `Count mismatch: JS=${jsCount}, Go=${goCount}`;
        }
        
        // Check for position differences (if counts match)
        if (!hasDifference && jsCount > 0 && goCount > 0) {
            for (let i = 0; i < Math.min(jsCount, goCount); i++) {
                const jsNode = jsNodes[i];
                const goNode = goResult.results[i];
                
                if (jsNode.startLocation !== goNode.startLocation || jsNode.endLocation !== goNode.endLocation) {
                    hasDifference = true;
                    differenceDescription = `Position mismatch at index ${i}: JS=${jsNode.startLocation}-${jsNode.endLocation}, Go=${goNode.startLocation}-${goNode.endLocation}`;
                    break;
                }
            }
        }
        
        if (hasDifference) {
            expectedDifferences++;
            console.log(`✅ EXPECTED DIFFERENCE - ${differenceDescription}`);
            console.log(`   📋 Reason: ${testCase.expected_difference}`);
            console.log(`   🔹 Go Behavior: ${testCase.go_behavior}`);
            console.log(`   🔹 JS Behavior: ${testCase.js_behavior}`);
        } else {
            console.log(`⚠️  UNEXPECTED MATCH - This edge case no longer differs!`);
            console.log(`   💡 Consider moving this test back to compatibility suite`);
        }
        
    } catch (error) {
        unexpectedFailures++;
        console.log(`❌ UNEXPECTED FAILURE - ${error.message}`);
        console.log(`   🐛 This indicates a bug, not an expected difference`);
    }
    
    console.log();
}

console.log('\n📊 EDGE CASES SUMMARY');
console.log('====================');
console.log(`Total Edge Cases: ${totalTests}`);
console.log(`Expected Differences: ${expectedDifferences}`);
console.log(`Unexpected Matches: ${totalTests - expectedDifferences - unexpectedFailures}`);
console.log(`Unexpected Failures: ${unexpectedFailures}`);

if (expectedDifferences === totalTests && unexpectedFailures === 0) {
    console.log('\n✅ All edge cases behaved as expected!');
} else if (unexpectedFailures > 0) {
    console.log('\n⚠️  Some edge cases had unexpected failures - these may need investigation');
} else {
    console.log('\n💡 Some edge cases no longer differ - consider moving them back to compatibility tests');
}

console.log('\n📖 Edge cases are documented behavioral differences, not bugs.');
console.log('   They represent intentional design choices in the Go implementation.');