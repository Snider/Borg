# Payment Integration Guide

This guide shows how to sell your encrypted `.smsg` content and deliver license keys (passwords) to customers using popular payment processors.

## Overview

The dapp.fm model is simple:

```
1. Customer pays via Stripe/Gumroad/PayPal
2. Payment processor triggers webhook or delivers digital product
3. Customer receives password (license key)
4. Customer downloads .smsg from your CDN/IPFS
5. Customer decrypts with password - done forever
```

No license servers, no accounts, no ongoing infrastructure.

## Stripe Integration

### Option 1: Stripe Payment Links (Easiest)

No code required - use Stripe's hosted checkout:

1. Create a Payment Link in Stripe Dashboard
2. Set up a webhook to email the password on successful payment
3. Host your `.smsg` file anywhere (CDN, IPFS, S3)

**Webhook endpoint (Node.js/Express):**

```javascript
const express = require('express');
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY);
const nodemailer = require('nodemailer');

const app = express();

// Your content passwords (store securely!)
const PRODUCTS = {
  'prod_ABC123': {
    name: 'My Album',
    password: 'PMVXogAJNVe_DDABfTmLYztaJAzsD0R7',
    downloadUrl: 'https://ipfs.io/ipfs/QmYourCID'
  }
};

app.post('/webhook', express.raw({type: 'application/json'}), async (req, res) => {
  const sig = req.headers['stripe-signature'];
  const endpointSecret = process.env.STRIPE_WEBHOOK_SECRET;

  let event;
  try {
    event = stripe.webhooks.constructEvent(req.body, sig, endpointSecret);
  } catch (err) {
    return res.status(400).send(`Webhook Error: ${err.message}`);
  }

  if (event.type === 'checkout.session.completed') {
    const session = event.data.object;
    const customerEmail = session.customer_details.email;
    const productId = session.metadata.product_id;
    const product = PRODUCTS[productId];

    if (product) {
      await sendLicenseEmail(customerEmail, product);
    }
  }

  res.json({received: true});
});

async function sendLicenseEmail(email, product) {
  const transporter = nodemailer.createTransport({
    // Configure your email provider
    service: 'gmail',
    auth: {
      user: process.env.EMAIL_USER,
      pass: process.env.EMAIL_PASS
    }
  });

  await transporter.sendMail({
    from: 'artist@example.com',
    to: email,
    subject: `Your License Key for ${product.name}`,
    html: `
      <h1>Thank you for your purchase!</h1>
      <p><strong>Download:</strong> <a href="${product.downloadUrl}">${product.name}</a></p>
      <p><strong>License Key:</strong> <code>${product.password}</code></p>
      <p><strong>How to play:</strong></p>
      <ol>
        <li>Download the .smsg file from the link above</li>
        <li>Go to <a href="https://demo.dapp.fm">demo.dapp.fm</a></li>
        <li>Click "Fan" tab, then "Unlock Licensed Content"</li>
        <li>Paste the file and enter your license key</li>
      </ol>
      <p>This is your permanent license - save this email!</p>
    `
  });
}

app.listen(3000);
```

### Option 2: Stripe Checkout Session (More Control)

```javascript
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY);

// Create checkout session
app.post('/create-checkout', async (req, res) => {
  const { productId } = req.body;

  const session = await stripe.checkout.sessions.create({
    payment_method_types: ['card'],
    line_items: [{
      price: 'price_ABC123', // Your Stripe price ID
      quantity: 1,
    }],
    mode: 'payment',
    success_url: 'https://yoursite.com/success?session_id={CHECKOUT_SESSION_ID}',
    cancel_url: 'https://yoursite.com/cancel',
    metadata: {
      product_id: productId
    }
  });

  res.json({ url: session.url });
});

// Success page - show license after payment
app.get('/success', async (req, res) => {
  const session = await stripe.checkout.sessions.retrieve(req.query.session_id);

  if (session.payment_status === 'paid') {
    const product = PRODUCTS[session.metadata.product_id];
    res.send(`
      <h1>Thank you!</h1>
      <p>Download: <a href="${product.downloadUrl}">${product.name}</a></p>
      <p>License Key: <code>${product.password}</code></p>
    `);
  } else {
    res.send('Payment not completed');
  }
});
```

## Gumroad Integration

Gumroad is perfect for artists - handles payments, delivery, and customer management.

### Setup

1. Create a Digital Product on Gumroad
2. Upload a text file or PDF containing the password
3. Set your `.smsg` download URL in the product description
4. Gumroad delivers the password file on purchase

### Product Setup

**Product Description:**
```
My Album - Encrypted Digital Download

After purchase, you'll receive:
1. A license key (in the download)
2. Download link for the .smsg file

How to play:
1. Download the .smsg file: https://ipfs.io/ipfs/QmYourCID
2. Go to https://demo.dapp.fm
3. Click "Fan" → "Unlock Licensed Content"
4. Enter your license key from the PDF
```

**Delivered File (license.txt):**
```
Your License Key: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7

Download your content: https://ipfs.io/ipfs/QmYourCID

This is your permanent license - keep this file safe!
The content works offline forever with this key.

Need help? Visit https://demo.dapp.fm
```

### Gumroad Ping (Webhook)

For automated delivery, use Gumroad's Ping feature:

```javascript
const express = require('express');
const app = express();

app.use(express.urlencoded({ extended: true }));

// Gumroad sends POST to this endpoint on sale
app.post('/gumroad-ping', (req, res) => {
  const {
    seller_id,
    product_id,
    email,
    full_name,
    purchaser_id
  } = req.body;

  // Verify it's from Gumroad (check seller_id matches yours)
  if (seller_id !== process.env.GUMROAD_SELLER_ID) {
    return res.status(403).send('Invalid seller');
  }

  const product = PRODUCTS[product_id];
  if (product) {
    // Send custom email with password
    sendLicenseEmail(email, product);
  }

  res.send('OK');
});
```

## PayPal Integration

### PayPal Buttons + IPN

```html
<!-- PayPal Buy Button -->
<form action="https://www.paypal.com/cgi-bin/webscr" method="post">
  <input type="hidden" name="cmd" value="_xclick">
  <input type="hidden" name="business" value="artist@example.com">
  <input type="hidden" name="item_name" value="My Album - Digital Download">
  <input type="hidden" name="item_number" value="album-001">
  <input type="hidden" name="amount" value="9.99">
  <input type="hidden" name="currency_code" value="USD">
  <input type="hidden" name="notify_url" value="https://yoursite.com/paypal-ipn">
  <input type="hidden" name="return" value="https://yoursite.com/thank-you">
  <input type="submit" value="Buy Now - $9.99">
</form>
```

**IPN Handler:**

```javascript
const express = require('express');
const axios = require('axios');

app.post('/paypal-ipn', express.urlencoded({ extended: true }), async (req, res) => {
  // Verify with PayPal
  const verifyUrl = 'https://ipnpb.paypal.com/cgi-bin/webscr';
  const verifyBody = 'cmd=_notify-validate&' + new URLSearchParams(req.body).toString();

  const response = await axios.post(verifyUrl, verifyBody);

  if (response.data === 'VERIFIED' && req.body.payment_status === 'Completed') {
    const email = req.body.payer_email;
    const itemNumber = req.body.item_number;
    const product = PRODUCTS[itemNumber];

    if (product) {
      await sendLicenseEmail(email, product);
    }
  }

  res.send('OK');
});
```

## Ko-fi Integration

Ko-fi is great for tips and single purchases.

### Setup

1. Enable "Commissions" or "Shop" on Ko-fi
2. Create a product with the license key in the thank-you message
3. Link to your .smsg download

**Ko-fi Thank You Message:**
```
Thank you for your purchase!

Your License Key: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7

Download: https://ipfs.io/ipfs/QmYourCID

Play at: https://demo.dapp.fm (Fan → Unlock Licensed Content)
```

## Serverless Options

### Vercel/Netlify Functions

No server needed - use serverless functions:

```javascript
// api/stripe-webhook.js (Vercel)
import Stripe from 'stripe';
import { Resend } from 'resend';

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY);
const resend = new Resend(process.env.RESEND_API_KEY);

export default async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).end();
  }

  const sig = req.headers['stripe-signature'];
  const event = stripe.webhooks.constructEvent(
    req.body,
    sig,
    process.env.STRIPE_WEBHOOK_SECRET
  );

  if (event.type === 'checkout.session.completed') {
    const session = event.data.object;

    await resend.emails.send({
      from: 'artist@yoursite.com',
      to: session.customer_details.email,
      subject: 'Your License Key',
      html: `
        <p>Download: <a href="https://ipfs.io/ipfs/QmYourCID">My Album</a></p>
        <p>License Key: <code>PMVXogAJNVe_DDABfTmLYztaJAzsD0R7</code></p>
      `
    });
  }

  res.json({ received: true });
}

export const config = {
  api: { bodyParser: false }
};
```

## Manual Workflow (No Code)

For artists who don't want to set up webhooks:

### Using Email

1. **Gumroad/Ko-fi**: Set product to require email
2. **Manual delivery**: Check sales daily, email passwords manually
3. **Template**:

```
Subject: Your License for [Album Name]

Hi [Name],

Thank you for your purchase!

Download: [IPFS/CDN link]
License Key: [password]

How to play:
1. Download the .smsg file
2. Go to demo.dapp.fm
3. Fan tab → Unlock Licensed Content
4. Enter your license key

Enjoy! This license works forever.

[Artist Name]
```

### Using Discord/Telegram

1. Sell via Gumroad (free tier)
2. Require customers join your Discord/Telegram
3. Bot or manual delivery of license keys
4. Community building bonus!

## Security Best Practices

### 1. One Password Per Product

Don't reuse passwords across products:

```javascript
const PRODUCTS = {
  'album-2024': { password: 'unique-key-1' },
  'album-2023': { password: 'unique-key-2' },
  'single-summer': { password: 'unique-key-3' }
};
```

### 2. Environment Variables

Never hardcode passwords in source:

```bash
# .env
ALBUM_2024_PASSWORD=PMVXogAJNVe_DDABfTmLYztaJAzsD0R7
STRIPE_SECRET_KEY=sk_live_...
```

### 3. Webhook Verification

Always verify webhooks are from the payment provider:

```javascript
// Stripe
stripe.webhooks.constructEvent(body, sig, secret);

// Gumroad
if (seller_id !== MY_SELLER_ID) reject();

// PayPal
verify with IPN endpoint
```

### 4. HTTPS Only

All webhook endpoints must use HTTPS.

## Pricing Strategies

### Direct Sale (Perpetual License)

- Customer pays once, owns forever
- Single password for all buyers
- Best for: Albums, films, books

### Time-Limited (Streaming/Rental)

Use dapp.fm Re-Key feature:

1. Encrypt master copy with master password
2. On purchase, re-key with customer-specific password + expiry
3. Deliver unique password per customer

```javascript
// On purchase webhook
const customerPassword = generateUniquePassword();
const expiry = Date.now() + (24 * 60 * 60 * 1000); // 24 hours

// Use WASM or Go to re-key
const customerVersion = await rekeyContent(masterSmsg, masterPassword, customerPassword, expiry);

// Deliver customer-specific file + password
```

### Tiered Access

Different passwords for different tiers:

```javascript
const TIERS = {
  'preview': { password: 'preview-key', expiry: '30s' },
  'rental': { password: 'rental-key', expiry: '7d' },
  'own': { password: 'perpetual-key', expiry: null }
};
```

## Example: Complete Stripe Setup

```bash
# 1. Create your content
go run ./cmd/mkdemo album.mp4 album.smsg
# Password: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7

# 2. Upload to IPFS
ipfs add album.smsg
# QmAlbumCID

# 3. Create Stripe product
# Dashboard → Products → Add Product
# Name: My Album
# Price: $9.99

# 4. Create Payment Link
# Dashboard → Payment Links → New
# Select your product
# Get link: https://buy.stripe.com/xxx

# 5. Set up webhook
# Dashboard → Developers → Webhooks → Add endpoint
# URL: https://yoursite.com/api/stripe-webhook
# Events: checkout.session.completed

# 6. Deploy webhook handler (Vercel example)
vercel deploy

# 7. Share payment link
# Fans click → Pay → Get email with password → Download → Play forever
```

## Resources

- [Stripe Webhooks](https://stripe.com/docs/webhooks)
- [Gumroad Ping](https://help.gumroad.com/article/149-ping)
- [PayPal IPN](https://developer.paypal.com/docs/ipn/)
- [Resend (Email API)](https://resend.com/)
- [Vercel Functions](https://vercel.com/docs/functions)
