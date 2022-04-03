#!/usr/bin/env node
///<reference types="@types/node"/>

import * as ts from 'typescript'
import * as path from 'path'
import * as readline from 'readline'

const diagnosticsHost: ts.FormatDiagnosticsHost = {
    getCurrentDirectory: () => ts.sys.getCurrentDirectory(),
    getNewLine: () => ts.sys.newLine,
    // Print filenames including their relativeRoot, so they can be located on
    // disk
    getCanonicalFileName: (f: string) => f,
}

function parseOptions(tsconfigPath: string) {
    const dir = path.dirname(path.resolve(process.cwd(), tsconfigPath))

    const { config, error } = ts.readConfigFile(tsconfigPath, ts.sys.readFile)
    if (error)
        throw new Error(
            tsconfigPath + ':' + ts.formatDiagnostic(error, diagnosticsHost)
        )
    const { errors, fileNames, projectReferences, options } =
        ts.parseJsonConfigFileContent(config, ts.sys, dir)
    if (errors && errors.length)
        throw new Error(
            tsconfigPath + ':' + ts.formatDiagnostics(errors, diagnosticsHost)
        )
    return {
        options,
        fileNames,
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
const rl = readline.createInterface({
    input: stdin,
})

interface Request {
    command: 'parseOptions'
    path: string
}

rl.on('line', (input) => {
    const req: Request = JSON.parse(input)
    let response: Object
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
