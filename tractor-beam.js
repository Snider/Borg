"use strict";
const fs = require('fs')
const fse = require('fs-extra');
const path = require('path');
const download = require('download-git-repo');
const rm = require('rimraf').sync
let root = 'src/assets/content'

let repos = [
    {project: 'chain', name: 'blockchain', url: 'https://gitlab.com/lthn.io/projects/chain/lethean.git#next'},
    {project: 'vpn', name: 'vpn', url: 'https://gitlab.com/lthn.io/projects/vpn/node.git'},
    {project: 'build', name: 'build', url: 'https://gitlab.com/lthn.io/projects/sdk/build.git#main'}
]

function parseDir(base) {
    let tree = []
    let meta = {}
    let dirs = fs.readdirSync(base)
    dirs.forEach((dir) => {
        if (fs.statSync(path.join(base, dir)).isDirectory()) {
            let dirUrl = path.join(base.replace(root, ''), dir)
            let docUrl = path.join(base.replace(root, ''), dirUrl.replace(dirUrl.substring(0, 4), '') + '/index.md')
            let httpUrl = dirUrl.replace(dirUrl.substring(0, 4), '')
            let children = parseDir(path.join(base, dir))
            meta[httpUrl.toString()] = dirUrl + '/index.md'
            tree.push({
                name: dir,
                url: httpUrl,
                doc: docUrl,
                children: children.tree
            })
            for (const [key, value] of Object.entries(children.meta)) {
                meta[key.toString()] = value
            }
        } else {
            if (dir !== 'index.md') {
                let fileUrl = path.join(base.replace(root, ''), dir.replace('.md', ''))
                let fileDoc = path.join(base.replace(root, ''), dir)
                let httpUrl = fileUrl.replace(fileUrl.substring(0, 4), '')
                tree.push({
                    name: dir.replace('.md', ''),
                    doc: fileDoc,
                    url: httpUrl
                })
                meta[httpUrl] = fileDoc
            }
        }
    })
    return { meta: meta, tree: tree }
}

console.log("Staring async Doc build")

repos.forEach((repo) => {
    rm(`docs/${repo.project}`)
    console.log(`Cloning ${repo.name} Project`)
    download(`direct:${repo.url}`, `brig/${repo.project}`, {
        clone: true, depth: 1
    }, function (err) {
        if (err) throw err
        console.log(`Assimilated ${repo.name} Repository`)

    })
})
