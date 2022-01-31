#!/usr/bin/env node

const rm = require('rimraf').sync;
const download = require('download-git-repo');
const { Octokit } = require("@octokit/core");



// Create a personal access token at https://github.com/settings/tokens/new?scopes=repo

const octokit = new Octokit({ auth: process.env.GH_TOKEN});
const target = process.env.BORG_TARGET ?  process.env.BORG_TARGET : process.argv[2] ? process.argv[2] : 'letheanVPN'
console.log(`Scanning For: ${target}`)

const MyActionOctokit = Octokit.plugin(
    require("@octokit/plugin-paginate-rest").paginateRest,
    require("@octokit/plugin-throttling").throttling,
    require("@octokit/plugin-retry").retry
).defaults({
    throttle: {
        onAbuseLimit: (retryAfter, options) => {
            /* ... */
        },
        onRateLimit: (retryAfter, options) => {
            /* ... */
        },
    },
    authStrategy: require("@octokit/auth-action").createActionAuth,
    userAgent: `my-octokit-action/v1.2.3`,
});

async function getList() {
    return await octokit.request(`GET /users/${target}/repos`, {
        type: "public",

    }).then((repos) => {
        repos.data.forEach((repo) => {
            console.log(`Brig Clean: ${repo.full_name}`)
            rm(`brig/${repo.full_name}`)
            download(`${repo.full_name}`, `brig/${repo.full_name}`, {
                // clone: true, depth: 1
            }, function (err) {
                if (err) throw err
                console.log(`Assimilated ${repo.full_name} Repository`)
            })
        })
    });
}


console.log(`We are the Borg. Your technological distinctiveness will be added to our own & contributed too`)

module.exports = getList();
