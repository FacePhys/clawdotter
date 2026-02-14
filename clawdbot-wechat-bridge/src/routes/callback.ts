import { FastifyInstance, FastifyRequest } from 'fastify';
import { sendTextMessage } from '../services/wechat-message.js';
import { getVMBinding } from '../services/redis.js';

/**
 * Callback payload from the OpenClaw VM.
 */
interface ClawdbotCallbackPayload {
    success: boolean;
    result?: string;
    error?: string;
    metadata?: {
        chunks?: number;
        thinking_time_ms?: number;
    };
}

interface CallbackParams {
    openid: string;
}

export async function callbackRoutes(fastify: FastifyInstance): Promise<void> {
    /**
     * POST /callback/:openid - Receive OpenClaw processing results
     * Called by the MicroVM after completing a task.
     */
    fastify.post<{
        Params: CallbackParams;
        Body: ClawdbotCallbackPayload;
    }>(
        '/callback/:openid',
        async (request, reply) => {
            const { openid } = request.params;
            const { success, result, error, metadata } = request.body;

            console.log(`Received callback for OpenID: ${openid}`, { success, metadata });

            // Verify the user still has a VM binding
            const binding = await getVMBinding(openid);
            if (!binding) {
                console.warn(`Callback received for unbound user: ${openid}`);
            }

            let messageContent: string;

            if (success) {
                messageContent = result || '✅ 任务已完成（无返回内容）';
            } else {
                messageContent = `❌ 处理失败：${error || '未知错误'}`;
            }

            if (metadata?.thinking_time_ms) {
                const seconds = (metadata.thinking_time_ms / 1000).toFixed(1);
                messageContent += `\n\n⏱️ 思考用时: ${seconds}s`;
            }

            // Send via Customer Service API
            const sent = await sendTextMessage(openid, messageContent);

            if (sent) {
                console.log(`Successfully sent response to ${openid}`);
                return reply.send({ ok: true });
            } else {
                console.error(`Failed to send response to ${openid}`);
                return reply.code(500).send({ ok: false, error: 'Failed to send WeChat message' });
            }
        }
    );

    /**
     * POST /callback/:openid/stream - Handle streaming responses
     */
    fastify.post<{
        Params: CallbackParams;
        Body: { chunk: string; done: boolean; chunk_index?: number };
    }>(
        '/callback/:openid/stream',
        async (request, reply) => {
            const { openid } = request.params;
            const { chunk, done, chunk_index } = request.body;

            if (done) {
                const sent = await sendTextMessage(openid, chunk);
                return reply.send({ ok: sent });
            }

            console.log(`Received stream chunk ${chunk_index || '?'} for ${openid}`);
            return reply.send({ ok: true, buffered: true });
        }
    );
}
