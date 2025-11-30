import { test, expect } from '@playwright/test';

test.describe('Chat Reload Test', () => {
  test('should preserve chat history after page reload', async ({ page }) => {
    // アプリケーションにアクセス
    await page.goto('http://localhost:3000');
    
    // チャット画面が表示されるまで待機
    await page.waitForSelector('text=/IT業界/', { timeout: 10000 });
    
    // セッションIDを取得
    const sessionId = await page.evaluate(() => {
      return localStorage.getItem('chat_session_id');
    });
    console.log('Session ID:', sessionId);
    expect(sessionId).toBeTruthy();
    
    // テストメッセージを送信
    const testMessage = 'E2Eテストメッセージ';
    const inputSelector = 'input[placeholder*="メッセージ"]';
    await page.waitForSelector(inputSelector, { timeout: 10000 });
    await page.fill(inputSelector, testMessage);
    await page.click('button[type="submit"]');
    
    // 送信したメッセージが表示されるまで待機
    await page.waitForSelector(`text=${testMessage}`, { timeout: 5000 });
    console.log('✓ Message sent successfully');
    
    // AIの応答を待つ
    await page.waitForTimeout(3000);
    
    // LocalStorageのキャッシュを確認
    const cachedMessages = await page.evaluate((sid) => {
      const cached = localStorage.getItem(`chat_cache_${sid}`);
      return cached ? JSON.parse(cached) : [];
    }, sessionId);
    console.log('Cached messages count:', cachedMessages.length);
    expect(cachedMessages.length).toBeGreaterThan(0);
    
    // ページをリロード
    console.log('Reloading page...');
    await page.reload();
    
    // チャット画面が再表示されるまで待機
    await page.waitForSelector('text=/IT業界/', { timeout: 10000 });
    
    // セッションIDが同じか確認
    const sessionIdAfterReload = await page.evaluate(() => {
      return localStorage.getItem('chat_session_id');
    });
    console.log('Session ID after reload:', sessionIdAfterReload);
    expect(sessionIdAfterReload).toBe(sessionId);
    
    // 短い待機時間を入れる
    await page.waitForTimeout(2000);
    
    // 送信したメッセージがまだ表示されているか確認
    const messageVisible = await page.locator(`text=${testMessage}`).isVisible();
    console.log('Message visible after reload:', messageVisible);
    
    if (!messageVisible) {
      // ページの内容をスクリーンショット
      await page.screenshot({ path: 'test-failure.png', fullPage: true });
      
      // LocalStorageの状態を確認
      const storageState = await page.evaluate(() => {
        return {
          sessionId: localStorage.getItem('chat_session_id'),
          cacheKeys: Object.keys(localStorage).filter(k => k.includes('chat_cache')),
        };
      });
      console.log('LocalStorage state:', storageState);
      
      // コンソールログを取得
      console.log('Page content:', await page.content());
    }
    
    expect(messageVisible).toBe(true);
    
    console.log('✅ Test passed: Chat history preserved after reload');
  });
  
  test('should show console logs during initialization', async ({ page }) => {
    // コンソールログを監視
    const logs: string[] = [];
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('[Frontend]')) {
        logs.push(text);
        console.log('📝', text);
      }
    });
    
    // アプリケーションにアクセス
    await page.goto('http://localhost:3000');
    
    // チャット画面が表示されるまで待機
    await page.waitForSelector('text=/IT業界/', { timeout: 10000 });
    await page.waitForTimeout(2000);
    
    // ログを確認
    console.log('\n=== Collected Logs ===');
    logs.forEach(log => console.log(log));
    console.log('======================\n');
    
    const hasSessionIdLog = logs.some(log => log.includes('Session ID:'));
    const hasInitLog = logs.some(log => log.includes('Initializing chat'));
    
    console.log('Has session ID log:', hasSessionIdLog);
    console.log('Has initialization log:', hasInitLog);
    
    expect(hasSessionIdLog).toBe(true);
    
    console.log('✅ Test passed: Console logs are working');
  });
});
