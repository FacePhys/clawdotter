# Clawdbot WeChat Channel Plugin

Connect your Clawdbot agent to WeChat Official Accounts.

**This plugin is part of the Clawdbot WeChat Integration Suite.**
For full source code, issues, and bridge deployment guide, please visit our GitHub repository:
👉 **[https://github.com/NannaOlympicBroadcast/clawdbot-wechat-plugin](https://github.com/NannaOlympicBroadcast/clawdbot-wechat-plugin)**

---

## 🚀 Installation

Install the plugin from the local directory:

```bash
openclaw plugins install ./clawdbot-plugin-webhook-server
```

Or from NPM:

```bash
openclaw plugins install @haiyanfengli-llc/webhook-server
```

## ⚙️ Configuration

You can configure the plugin via the GUI (Recommended) or by editing the `clawdbot.json` file directly.

### GUI Configuration (Recommended)

1.  Navigate to your Clawdbot Dashboard: `http(s)://<gatewayurl>/config`
2.  Scroll down the sidebar and click on **Plugins**.
3.  Select the **All** tab.
4.  Scroll down to the **WeChat** card to configure the settings.

### Manual Configuration

Add the following configuration to your Clawdbot `clawdbot.json`:

```json
{
  "channels": {
    "wechat": {
      "enabled": true,
      "config": {
        "callbackUrl": "http://<bridge-host>:3000/callback"
      }
    }
  },
  "plugins": {
    "entries": {
      "webhook-server": {
        "enabled": true
      }
    }
  }
}
```

## 🔗 Architecture

This plugin requires the **WeChat Bridge** service to function.
The bridge handles the communication with WeChat servers and forwards messages to this plugin.

1.  **WeChat** sends message to **Bridge**.
2.  **Bridge** forwards message to **Clawdbot Plugin**.
3.  **Clawdbot Agent** processes message.
4.  **Clawdbot Plugin** sends reply back to **Bridge**.
5.  **Bridge** sends reply to **WeChat**.

Please refer to the [GitHub Repository](https://github.com/NannaOlympicBroadcast/clawdbot-wechat-plugin) for instructions on how to deploy the Bridge.

## 📋 Requirements

*   Clawdbot v0.5.0 or later
*   Self-hosted WeChat Bridge
*   WeChat Service Account (服务号) or verified Subscription Account (认证订阅号)

## 🤝 Commercial Support

For commercial usage, verified builds, or enterprise support, please contact:
📧 **nomorelighthouse@gmail.com**
