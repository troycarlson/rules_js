#!/usr/bin/env node
'use strict'
///<reference types="@types/node"/>
exports.__esModule = true
var ts = require('typescript')
var path = require('path')
var readline = require('readline')
var diagnosticsHost = {
    getCurrentDirectory: function () {
        return ts.sys.getCurrentDirectory()
    },
    getNewLine: function () {
        return ts.sys.newLine
    },
    // Print filenames including their relativeRoot, so they can be located on
    // disk
    getCanonicalFileName: function (f) {
        return f
    },
}
function parseOptions(tsconfigPath) {
    var dir = path.dirname(path.resolve(process.cwd(), tsconfigPath))
    var _a = ts.readConfigFile(tsconfigPath, ts.sys.readFile),
        config = _a.config,
        error = _a.error
    if (error)
        throw new Error(
            tsconfigPath + ':' + ts.formatDiagnostic(error, diagnosticsHost)
        )
    var _b = ts.parseJsonConfigFileContent(config, ts.sys, dir),
        errors = _b.errors,
        fileNames = _b.fileNames,
        projectReferences = _b.projectReferences,
        options = _b.options
    if (errors && errors.length)
        throw new Error(
            tsconfigPath + ':' + ts.formatDiagnostics(errors, diagnosticsHost)
        )
    return {
        options: options,
        fileNames: fileNames,
    }
    // const host: ts.LanguageServiceHost = {
    //     getCompilationSettings: () => options,
    //     getScriptFileNames: () => fileNames,
    //     getScriptVersion: (filename: string) => "1",
    //     getScriptSnapshot
    // }
    // ts.createLanguageService(host)
}
var stdin = process.stdin
stdin.setEncoding('utf8')
var rl = readline.createInterface({
    input: stdin,
})
rl.on('line', function (input) {
    var req = JSON.parse(input)
    var response
    switch (req.command) {
        case 'parseOptions':
            response = parseOptions(req.path)
            break
        default:
            throw new Error('unsupported command ' + req.command)
    }
    console.log(JSON.stringify(response))
})
stdin.on('end', function () {
    process.exitCode = 0
})
