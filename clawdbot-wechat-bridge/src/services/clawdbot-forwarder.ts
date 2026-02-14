import axios from 'axios';
import { getConfig } from '../config.js';
import type { WeChatMessage } from '../utils/xml-parser.js';
import type { VMBinding } from './redis.js';

/**
 * Payload sent to the OpenClaw webhook inside the MicroVM.
 */
export interface ClawdbotWebhookPayload {
    task: string;
    callback_url: string;
    metadata: {
        openid: string;
        msg_type: string;
        msg_id?: string;
        timestamp: number;
    };
}

/**
 * Forward a WeChat message to the user's OpenClaw VM.
 * Routes to the internal VPC IP — no auth needed on private network.
 */
export function forwardToClawdbot(
    message: WeChatMessage,
    binding: VMBinding
): void {
    doForward(message, binding).catch((error) => {
        console.error(`Failed to forward message to VM:`, error);
    });
}

/**
 * Internal forwarding implementation.
 */
async function doForward(
    message: WeChatMessage,
    binding: VMBinding
): Promise<void> {
    const config = getConfig();

    // Determine the task content based on message type
    let task: string;
    switch (message.MsgType) {
        case 'text':
            task = message.Content || '';
            break;
        case 'voice':
            task = message.Recognition || '[语音消息，无法识别]';
            break;
        case 'image':
            task = `[图片消息] ${message.PicUrl || ''}`;
            break;
        case 'location':
            task = `[位置消息] 经度: ${message.Location_Y}, 纬度: ${message.Location_X}, ${message.Label || ''}`;
            break;
        case 'link':
            task = `[链接消息] ${message.Title || ''}\n${message.Description || ''}\n${message.Url || ''}`;
            break;
        default:
            task = `[${message.MsgType}消息]`;
    }

    const callbackUrl = `${config.bridge.baseUrl}/callback/${message.FromUserName}`;

    const payload: ClawdbotWebhookPayload = {
        task,
        callback_url: callbackUrl,
        metadata: {
            openid: message.FromUserName,
            msg_type: message.MsgType,
            msg_id: message.MsgId,
            timestamp: message.CreateTime,
        },
    };

    // Route to internal VPC IP — no auth token needed
    console.log(`Forwarding message to VM: ${binding.webhookUrl}`);

    const response = await axios.post(binding.webhookUrl, payload, {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: 10000,
    });

    console.log(`VM responded with status: ${response.status}`);
}
