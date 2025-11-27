export interface ChatRequest {
    user_id: number
    session_id: string
    message: string
    industry_id: number
    job_category_id: number
}

export interface ChatResponse {
    response: string
    question_weight_id?: number
    is_complete: boolean
    total_questions: number
    answered_questions: number
    current_scores?: Array<{
        id: number
        user_id: number
        session_id: string
        category: string
        score: number
        created_at: string
        updated_at: string
    }>
}

export interface ChatHistory {
    id: number
    session_id: string
    user_id: number
    role: "user" | "assistant"
    content: string
    question_weight_id?: number
    created_at: string
}

export async function sendChatMessage(request: ChatRequest): Promise<ChatResponse> {
    const response = await fetch('/api/chat', {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(request),
    })

    if (!response.ok) {
        throw new Error(`Chat API error: ${response.statusText}`)
    }

    return response.json()
}

export async function getChatHistory(sessionId: string): Promise<ChatHistory[]> {
    const response = await fetch(`/api/chat/history?session_id=${sessionId}`)

    if (!response.ok) {
        throw new Error(`History API error: ${response.statusText}`)
    }

    return response.json()
}

export async function getUserScores(userId: number, sessionId: string) {
    const response = await fetch(`/api/chat/scores?user_id=${userId}&session_id=${sessionId}`)

    if (!response.ok) {
        throw new Error(`Scores API error: ${response.statusText}`)
    }

    return response.json()
}

export async function getRecommendations(userId: number, sessionId: string, limit = 5) {
    const response = await fetch(
        `/api/chat/recommendations?user_id=${userId}&session_id=${sessionId}&limit=${limit}`,
    )

    if (!response.ok) {
        throw new Error(`Recommendations API error: ${response.statusText}`)
    }

    return response.json()
}

export async function sendMessage(message: string): Promise<{ message: string }> {
    const sessionId = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`
    
    try {
        const response = await sendChatMessage({
            user_id: 1,
            session_id: sessionId,
            message,
            industry_id: 1,
            job_category_id: 1,
        })
        
        return {
            message: response.response || 'メッセージを受信しました。',
        }
    } catch (error) {
        throw error
    }
}
