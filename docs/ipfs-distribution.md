# IPFS Distribution Guide

This guide explains how to distribute your encrypted `.smsg` content via IPFS (InterPlanetary File System) for permanent, decentralized hosting.

## Why IPFS?

IPFS is ideal for dapp.fm content because:

- **Permanent links** - Content-addressed (CID) means the URL never changes
- **No hosting costs** - Pin with free services or self-host
- **Censorship resistant** - No single point of failure
- **Global CDN** - Content served from nearest peer
- **Perfect for archival** - Your content survives even if you disappear

Combined with password-as-license, IPFS creates truly permanent media distribution:

```
Artist uploads to IPFS → Fan downloads from anywhere → Password unlocks forever
```

## Quick Start

### 1. Install IPFS

**macOS:**
```bash
brew install ipfs
```

**Linux:**
```bash
wget https://dist.ipfs.tech/kubo/v0.24.0/kubo_v0.24.0_linux-amd64.tar.gz
tar xvfz kubo_v0.24.0_linux-amd64.tar.gz
sudo mv kubo/ipfs /usr/local/bin/
```

**Windows:**
Download from https://dist.ipfs.tech/#kubo

### 2. Initialize and Start

```bash
ipfs init
ipfs daemon
```

### 3. Add Your Content

```bash
# Create your encrypted content first
go run ./cmd/mkdemo my-album.mp4 my-album.smsg

# Add to IPFS
ipfs add my-album.smsg
# Output: added QmX...abc my-album.smsg

# Your content is now available at:
# - Local: http://localhost:8080/ipfs/QmX...abc
# - Gateway: https://ipfs.io/ipfs/QmX...abc
```

## Distribution Workflow

### For Artists

```bash
# 1. Package your media
go run ./cmd/mkdemo album.mp4 album.smsg
# Save the password: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7

# 2. Add to IPFS
ipfs add album.smsg
# added QmYourContentCID album.smsg

# 3. Pin for persistence (choose one):

# Option A: Pin locally (requires running node)
ipfs pin add QmYourContentCID

# Option B: Use Pinata (free tier: 1GB)
curl -X POST "https://api.pinata.cloud/pinning/pinByHash" \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{"hashToPin": "QmYourContentCID"}'

# Option C: Use web3.storage (free tier: 5GB)
# Upload at https://web3.storage

# 4. Share with fans
# CID: QmYourContentCID
# Password: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7
# Gateway URL: https://ipfs.io/ipfs/QmYourContentCID
```

### For Fans

```bash
# Download via any gateway
curl -o album.smsg https://ipfs.io/ipfs/QmYourContentCID

# Or via local node (faster if running)
ipfs get QmYourContentCID -o album.smsg

# Play with password in browser demo or native app
```

## IPFS Gateways

Public gateways for sharing (no IPFS node required):

| Gateway | URL Pattern | Notes |
|---------|-------------|-------|
| ipfs.io | `https://ipfs.io/ipfs/{CID}` | Official, reliable |
| dweb.link | `https://{CID}.ipfs.dweb.link` | Subdomain style |
| cloudflare | `https://cloudflare-ipfs.com/ipfs/{CID}` | Fast, cached |
| w3s.link | `https://{CID}.ipfs.w3s.link` | web3.storage |
| nftstorage.link | `https://{CID}.ipfs.nftstorage.link` | NFT.storage |

**Example URLs for CID `QmX...abc`:**
```
https://ipfs.io/ipfs/QmX...abc
https://QmX...abc.ipfs.dweb.link
https://cloudflare-ipfs.com/ipfs/QmX...abc
```

## Pinning Services

Content on IPFS is only available while someone is hosting it. Use pinning services for persistence:

### Free Options

| Service | Free Tier | Link |
|---------|-----------|------|
| Pinata | 1 GB | https://pinata.cloud |
| web3.storage | 5 GB | https://web3.storage |
| NFT.storage | Unlimited* | https://nft.storage |
| Filebase | 5 GB | https://filebase.com |

*NFT.storage is designed for NFT data but works for any content.

### Pin via CLI

```bash
# Pinata
export PINATA_JWT="your-jwt-token"
curl -X POST "https://api.pinata.cloud/pinning/pinByHash" \
  -H "Authorization: Bearer $PINATA_JWT" \
  -H "Content-Type: application/json" \
  -d '{"hashToPin": "QmYourCID", "pinataMetadata": {"name": "my-album.smsg"}}'

# web3.storage (using w3 CLI)
npm install -g @web3-storage/w3cli
w3 login your@email.com
w3 up my-album.smsg
```

## Integration with Demo Page

The demo page can load content directly from IPFS gateways:

```javascript
// In the demo page, use gateway URL
const ipfsCID = 'QmYourContentCID';
const gatewayUrl = `https://ipfs.io/ipfs/${ipfsCID}`;

// Fetch and decrypt
const response = await fetch(gatewayUrl);
const bytes = new Uint8Array(await response.arrayBuffer());
const msg = await BorgSMSG.decryptBinary(bytes, password);
```

Or use the Fan tab with the IPFS gateway URL directly.

## Best Practices

### 1. Always Pin Your Content

IPFS garbage-collects unpinned content. Always pin important files:

```bash
ipfs pin add QmYourCID
# Or use a pinning service
```

### 2. Use Multiple Pins

Pin with 2-3 services for redundancy:

```bash
# Pin locally
ipfs pin add QmYourCID

# Also pin with Pinata
curl -X POST "https://api.pinata.cloud/pinning/pinByHash" ...

# And web3.storage as backup
w3 up my-album.smsg
```

### 3. Share CID + Password Separately

```
Download: https://ipfs.io/ipfs/QmYourCID
License: [sent via email/DM after purchase]
```

### 4. Use IPNS for Updates (Optional)

IPNS lets you update content while keeping the same URL:

```bash
# Create IPNS name
ipfs name publish QmYourCID
# Published to k51...xyz

# Your content is now at:
# https://ipfs.io/ipns/k51...xyz

# Update to new version later:
ipfs name publish QmNewVersionCID
```

## Example: Full Album Release

```bash
# 1. Create encrypted album
go run ./cmd/mkdemo my-album.mp4 my-album.smsg
# Password: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7

# 2. Add to IPFS
ipfs add my-album.smsg
# added QmAlbumCID my-album.smsg

# 3. Pin with multiple services
ipfs pin add QmAlbumCID
w3 up my-album.smsg

# 4. Create release page
cat > release.html << 'EOF'
<!DOCTYPE html>
<html>
<head><title>My Album - Download</title></head>
<body>
  <h1>My Album</h1>
  <p>Download: <a href="https://ipfs.io/ipfs/QmAlbumCID">IPFS</a></p>
  <p>After purchase, you'll receive your license key via email.</p>
  <p><a href="https://demo.dapp.fm">Play with license key</a></p>
</body>
</html>
EOF

# 5. Host release page on IPFS too!
ipfs add release.html
# added QmReleaseCID release.html
# Share: https://ipfs.io/ipfs/QmReleaseCID
```

## Troubleshooting

### Content Not Loading

1. **Check if pinned**: `ipfs pin ls | grep QmYourCID`
2. **Try different gateway**: Some gateways cache slowly
3. **Check daemon running**: `ipfs swarm peers` should show peers

### Slow Downloads

1. Use a faster gateway (cloudflare-ipfs.com is often fastest)
2. Run your own IPFS node for direct access
3. Pre-warm gateways by accessing content once

### CID Changed After Re-adding

IPFS CIDs are content-addressed. If you modify the file, the CID changes. For the same content, the CID is always identical.

## Resources

- [IPFS Documentation](https://docs.ipfs.tech/)
- [Pinata Docs](https://docs.pinata.cloud/)
- [web3.storage Docs](https://web3.storage/docs/)
- [IPFS Gateway Checker](https://ipfs.github.io/public-gateway-checker/)
