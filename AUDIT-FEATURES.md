
# Feature Audit: dapp-fm vs. Competitors

This audit compares the features of `dapp-fm` against popular data collection and archiving tools.

## Feature Comparison Matrix

| Feature                      | dapp-fm | wget/curl | HTTrack | ArchiveBox | SingleFile | rclone |
| ---------------------------- | ------- | --------- | ------- | ---------- | ---------- | ------ |
| **General**                  |         |           |         |            |            |        |
| Target                       | Websites, Git Repos, PWAs | Files, Websites | Websites | Websites | Webpages | Cloud Storage |
| Output Format                | datanode, tim, trix, stim | Files | HTML | HTML, WARC, etc. | HTML | Files |
| **Website Archiving**        |         |           |         |            |            |        |
| Recursive Download           | Yes     | Yes       | Yes     | Yes        | No         | N/A    |
| Asset Capture (JS, CSS, etc.)| Yes     | Yes       | Yes     | Yes        | Yes        | N/A    |
| MHTML/WARC Output            | No      | No        | No      | Yes        | No         | N/A    |
| Single Page Archive          | Yes     | Yes       | Yes     | Yes        | Yes        | N/A    |
| **Data Sources**             |         |           |         |            |            |        |
| Git Repositories             | Yes     | No        | No      | Yes        | No         | No     |
| GitHub Releases              | Yes     | No        | No      | No         | No         | No     |
| Progressive Web Apps (PWAs)  | Yes     | No        | No      | No         | No         | No     |
| **Storage & Backend**        |         |           |         |            |            |        |
| Cloud Storage Sync           | No      | No        | No      | No         | No         | Yes    |
| **Advanced Features**        |         |           |         |            |            |        |
| Headless Browser             | No      | No        | No      | Yes        | Yes        | N/A    |
| Authentication               | No      | Yes       | Yes     | Yes        | No         | Yes    |
| Rate Limiting                | No      | Yes       | Yes     | Yes        | No         | Yes    |
| Filtering (Include/Exclude)  | No      | Yes       | Yes     | Yes        | No         | Yes    |
| Scheduling                   | No      | No        | No      | Yes        | No         | No     |
| **Usability**                |         |           |         |            |            |        |
| CLI Interface                | Yes     | Yes       | Yes     | Yes        | No         | Yes    |
| GUI Interface                | No      | No        | Yes     | Yes        | Yes (Browser Ext) | No |

## Analysis

### Missing Core Features

*   **Headless Browser Rendering:** `dapp-fm` doesn't render pages in a headless browser, which means it may not capture content from single-page applications (SPAs) or websites that rely heavily on JavaScript.
*   **Standard Archive Formats:** The tool doesn't export to standard formats like WARC or MHTML, which are widely used in web archiving.
*   **Authentication and Rate Limiting:** `dapp-fm` lacks built-in support for handling websites that require logins or have rate limits.
*   **Cloud Storage Integration:** Unlike `rclone`, `dapp-fm` cannot sync archives to various cloud storage providers.
*   **Scheduling:** There's no built-in mechanism for scheduling recurring captures.

### Competitive Advantages

*   **Diverse Data Sources:** `dapp-fm`'s ability to collect not just websites but also Git repositories and Progressive Web Apps gives it a unique advantage.
*   **Proprietary Archiving Formats:** The `.trix` and `.stim` formats, with their encryption and compression capabilities, offer a secure and efficient way to store and share archives.
*   **Simplicity and Focus:** `dapp-fm` has a clear focus on collecting specific types of online resources and packaging them into a portable format.

### Integration Opportunities

*   **Browser Extension:** A browser extension, similar to `SingleFile`, could streamline the process of capturing single pages.
*   **Cloud Storage Providers:** Integrating with services like Amazon S3, Google Cloud Storage, or Dropbox would make it easier for users to store and manage their archives.
*   **CI/CD Integration:** `dapp-fm` could be integrated into CI/CD pipelines to automatically archive websites or applications after deployment.

### User Workflow Gaps

*   **No GUI:** The lack of a graphical interface makes `dapp-fm` less accessible to non-technical users.
*   **Limited Configuration:** The tool offers limited configuration options for things like filtering content, setting user agents, or handling cookies.
*   **Post-Archival Management:** `dapp-fm` doesn't provide any tools for managing, searching, or viewing archives after they've been created.
