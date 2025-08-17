const fs = require('fs');
const { JSDOM } = require('jsdom');
const { execSync } = require('child_process');
const path = require('path');

// Load test cases
const testcases = JSON.parse(fs.readFileSync('./shared/testcases.json', 'utf8'));
const extendedTestcases = JSON.parse(fs.readFileSync('./shared/extended_testcases.json', 'utf8'));

// Failing test indices based on the comprehensive test output
const failingTestIndices = [
    35,  // Empty elements
    38,  // Ancestor-or-self axis
    48,  // String functions combination
    54,  // Complex table navigation  
    59,  // Empty vs non-empty elements
    62,  // Position in filtered set
    66,  // Class list manipulation
    67,  // Document structure validation
    71,  // Substring length comparison
    72   // Substring edge cases
];

// Combine all tests
const allTests = [...testcases, ...extendedTestcases];

function evaluateJSXPath(html, xpath) {
    const dom = new JSDOM(html);
    const document = dom.window.document;
    const results = [];
    
    try {
        const xpathResult = document.evaluate(
            xpath,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        for (let i = 0; i < xpathResult.snapshotLength; i++) {
            const node = xpathResult.snapshotItem(i);
            results.push({
                nodeName: node.nodeName.toLowerCase(),
                textContent: node.textContent || '',
                attributes: node.attributes ? 
                    Array.from(node.attributes).reduce((acc, attr) => {
                        acc[attr.name] = attr.value;
                        return acc;
                    }, {}) : {}
            });
        }
    } catch (e) {
        console.error('JS XPath evaluation error:', e.message);
    }
    
    return results;
}

function evaluateGoXPath(html, xpath) {
    // Create temp files
    const tempHtml = `temp_debug_${Date.now()}.html`;
    const tempXPath = `temp_debug_${Date.now()}.xpath`;
    
    fs.writeFileSync(tempHtml, html);
    fs.writeFileSync(tempXPath, xpath);
    
    try {
        const output = execSync(`../xpath-go --html ${tempHtml} --xpath ${tempXPath}`, {
            encoding: 'utf8',
            cwd: __dirname
        });
        
        // Clean up temp files
        fs.unlinkSync(tempHtml);
        fs.unlinkSync(tempXPath);
        
        // Parse output
        const lines = output.trim().split('\n').filter(line => line);
        if (lines.length === 0 || lines[0] === 'No matching nodes found') {
            return [];
        }
        
        return lines.map(line => {
            const match = line.match(/^(\w+)(?:\[([^\]]*)\])?:\s*(.*)$/);
            if (!match) return null;
            
            const [, nodeName, attrs, text] = match;
            const attributes = {};
            
            if (attrs) {
                attrs.split(',').forEach(attr => {
                    const [key, value] = attr.split('=');
                    if (key && value) {
                        attributes[key.trim()] = value.replace(/['"]/g, '').trim();
                    }
                });
            }
            
            return {
                nodeName: nodeName.toLowerCase(),
                textContent: text || '',
                attributes
            };
        }).filter(Boolean);
    } catch (e) {
        console.error('Go XPath evaluation error:', e.message);
        // Clean up temp files on error
        if (fs.existsSync(tempHtml)) fs.unlinkSync(tempHtml);
        if (fs.existsSync(tempXPath)) fs.unlinkSync(tempXPath);
        return [];
    }
}

console.log('🔍 DETAILED ANALYSIS OF FAILING TESTS');
console.log('=' .repeat(50));

failingTestIndices.forEach((index, i) => {
    const test = allTests[index];
    if (!test) {
        console.log(`\n❌ Test ${index} not found`);
        return;
    }
    
    console.log(`\n[${i + 1}/${failingTestIndices.length}] Test #${index + 1}: ${test.name}`);
    console.log(`XPath: ${test.xpath}`);
    console.log('-'.repeat(50));
    
    // Evaluate with both engines
    const jsResults = evaluateJSXPath(test.html, test.xpath);
    const goResults = evaluateGoXPath(test.html, test.xpath);
    
    console.log(`\n📊 Results:`);
    console.log(`JS Results (${jsResults.length} nodes):`);
    jsResults.forEach((node, idx) => {
        console.log(`  ${idx + 1}. ${node.nodeName}${Object.keys(node.attributes).length > 0 ? `[${Object.entries(node.attributes).map(([k,v]) => `${k}="${v}"`).join(', ')}]` : ''}: "${node.textContent.substring(0, 50)}${node.textContent.length > 50 ? '...' : ''}"`);
    });
    
    console.log(`\nGo Results (${goResults.length} nodes):`);
    goResults.forEach((node, idx) => {
        console.log(`  ${idx + 1}. ${node.nodeName}${Object.keys(node.attributes).length > 0 ? `[${Object.entries(node.attributes).map(([k,v]) => `${k}="${v}"`).join(', ')}]` : ''}: "${node.textContent.substring(0, 50)}${node.textContent.length > 50 ? '...' : ''}"`);
    });
    
    // Analyze difference
    console.log(`\n🔎 Analysis:`);
    if (jsResults.length !== goResults.length) {
        console.log(`  ⚠️ Count mismatch: JS=${jsResults.length}, Go=${goResults.length}`);
    } else {
        for (let j = 0; j < jsResults.length; j++) {
            if (jsResults[j].nodeName !== goResults[j].nodeName) {
                console.log(`  ⚠️ Node name mismatch at index ${j}: JS="${jsResults[j].nodeName}", Go="${goResults[j].nodeName}"`);
            }
            if (jsResults[j].textContent !== goResults[j].textContent) {
                console.log(`  ⚠️ Text content mismatch at index ${j}`);
                console.log(`     JS: "${jsResults[j].textContent}"`);
                console.log(`     Go: "${goResults[j].textContent}"`);
            }
        }
    }
    
    // Show HTML snippet for context
    console.log(`\n📄 HTML Context:`);
    console.log(test.html.substring(0, 300) + (test.html.length > 300 ? '...' : ''));
});