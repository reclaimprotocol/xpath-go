'use strict';

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');
const { getHtmlFilePath } = require('./test_cases');

class GoOracle {
    constructor(testsDirectory, options = {}) {
        this.testsDirectory = testsDirectory;
        this.trace = Boolean(options.trace);
        this.binaryPath = null;
        this.cleanupRegistered = false;
    }

    evaluate(xpath, html, contentsOnly = false, testCase = null) {
        let temporaryHtmlPath = null;
        let xpathPath = null;
        try {
            const existingHtmlPath = testCase
                ? getHtmlFilePath(testCase, this.testsDirectory)
                : null;
            const suffix = `${process.pid}_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
            const htmlPath = existingHtmlPath || path.join(this.testsDirectory, `temp_html_${suffix}.html`);
            xpathPath = path.join(this.testsDirectory, `temp_xpath_${suffix}.txt`);

            if (!existingHtmlPath) {
                temporaryHtmlPath = htmlPath;
                fs.writeFileSync(htmlPath, html);
            }
            fs.writeFileSync(xpathPath, xpath);

            const args = [htmlPath, xpathPath];
            if (this.trace) args.push('--trace');
            if (contentsOnly) args.push('--contents-only');
            if (testCase && testCase.charset) args.push('--charset', testCase.charset);

            const output = this.executeWithRetry(this.ensureBinary(), args);
            const result = JSON.parse(output.trim());
            if (result.error) return failure(result.error);
            return {
                success: true,
                results: result.results || [],
                count: result.count || 0,
                error: null
            };
        } catch (error) {
            return failure(error.message);
        } finally {
            removeFile(temporaryHtmlPath);
            removeFile(xpathPath);
        }
    }

    ensureBinary() {
        if (this.binaryPath) return this.binaryPath;

        const goDirectory = path.join(this.testsDirectory, 'go');
        this.binaryPath = path.join(goDirectory, `.comprehensive_compare_${process.pid}`);
        execFileSync('go', ['build', '-o', this.binaryPath, 'main.go'], {
            cwd: goDirectory,
            encoding: 'utf8',
            timeout: 120000,
            maxBuffer: 1024 * 1024
        });
        if (!this.cleanupRegistered) {
            process.once('exit', () => this.dispose());
            this.cleanupRegistered = true;
        }
        return this.binaryPath;
    }

    executeWithRetry(binaryPath, args) {
        let lastError;
        for (let attempt = 0; attempt < 2; attempt++) {
            try {
                return execFileSync(binaryPath, args, {
                    encoding: 'utf8',
                    timeout: 30000,
                    maxBuffer: 1024 * 1024
                });
            } catch (error) {
                lastError = error;
            }
        }
        throw lastError;
    }

    dispose() {
        removeFile(this.binaryPath);
        this.binaryPath = null;
    }
}

function removeFile(filePath) {
    if (!filePath) return;
    try {
        fs.unlinkSync(filePath);
    } catch (error) {
        if (error.code !== 'ENOENT') throw error;
    }
}

function failure(error) {
    return { success: false, results: [], count: 0, error };
}

module.exports = { GoOracle };
