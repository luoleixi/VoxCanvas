import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Mic, RadioTower, Sparkles, Loader2 } from 'lucide-react';
import './styles.css';

const VOICE_ENDPOINT = import.meta.env.VITE_VOICE_ENDPOINT || '/api/voice';
const RESTART_DELAY_MS = 350;

function normalizeImageSource(content) {
  if (!content) return '';
  if (content.startsWith('data:')) return content;
  if (content.startsWith('/9j/')) return `data:image/jpeg;base64,${content}`;
  if (content.startsWith('R0lGOD')) return `data:image/gif;base64,${content}`;
  if (content.startsWith('UklGR')) return `data:image/webp;base64,${content}`;
  return `data:image/png;base64,${content}`;
}

function App() {
  const [isListening, setIsListening] = useState(false);
  const [speechSupported, setSpeechSupported] = useState(true);
  const [liveTranscript, setLiveTranscript] = useState('');
  const [promptText, setPromptText] = useState('');
  const [imageSrc, setImageSrc] = useState('');
  const [statusText, setStatusText] = useState('正在准备语音识别...');
  const shouldRestartRef = useRef(true);
  const restartTimerRef = useRef(null);

  // 增加波纹条数量，让动画更细腻
  const waveformBars = useMemo(
    () => Array.from({ length: 24 }, (_, index) => ({ id: index, delay: `${index * 0.05}s` })),
    [],
  );

  useEffect(() => {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;

    if (!SpeechRecognition) {
      setSpeechSupported(false);
      setStatusText('当前浏览器不支持 Web Speech API，请使用 Chrome 体验。');
      return undefined;
    }

    shouldRestartRef.current = true;
    const recognition = new SpeechRecognition();
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = 'zh-CN';

    const startRecognition = () => {
      if (!shouldRestartRef.current) return;
      try {
        recognition.start();
      } catch (error) {
        if (error.name !== 'InvalidStateError') {
          console.warn('语音识别启动失败:', error);
          setStatusText('语音识别启动失败，请刷新后重试。');
        }
      }
    };

    recognition.onstart = () => {
      setIsListening(true);
      setStatusText('正在倾听...');
    };

    recognition.onresult = (event) => {
      let interimTranscript = '';
      let finalTranscript = '';

      for (let index = event.resultIndex; index < event.results.length; index += 1) {
        const transcript = event.results[index][0].transcript;
        if (event.results[index].isFinal) {
          finalTranscript += transcript;
        } else {
          interimTranscript += transcript;
        }
      }

      const sentence = finalTranscript.trim();
      if (sentence) {
        setLiveTranscript(sentence);
        handleFinalSentence(sentence);
      } else if (interimTranscript) {
        setLiveTranscript(interimTranscript);
      }
    };

    recognition.onerror = (event) => {
      if (event.error === 'aborted') return;
      if (event.error === 'no-speech') {
        setStatusText('正在倾听...');
        return;
      }
      if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
        shouldRestartRef.current = false;
        setIsListening(false);
        setStatusText('麦克风权限被拒绝');
        return;
      }
      if (event.error === 'network') {
        setStatusText('语音服务网络异常，正在尝试恢复...');
        return;
      }
      setStatusText(`语音识别异常：${event.error}`);
    };

    recognition.onend = () => {
      setIsListening(false);
      if (!shouldRestartRef.current) return;
      setStatusText('正在重新连接语音识别...');
      window.clearTimeout(restartTimerRef.current);
      restartTimerRef.current = window.setTimeout(startRecognition, RESTART_DELAY_MS);
    };

    startRecognition();

    return () => {
      shouldRestartRef.current = false;
      window.clearTimeout(restartTimerRef.current);
      recognition.onstart = null;
      recognition.onresult = null;
      recognition.onerror = null;
      recognition.onend = null;
      recognition.abort();
    };
  }, []);

  async function handleFinalSentence(sentence) {
    setStatusText('正在解析语义并生成图象...');
    try {
      const response = await fetch(VOICE_ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sentences: sentence }),
      });

      const payload = await response.json();
      if (!response.ok || payload.code !== 200) throw new Error(payload.msg || `请求失败：${response.status}`);

      if (payload.data?.op === 'requirement') setPromptText(payload.data.content || '');
      if (payload.data?.op === 'order') setImageSrc(normalizeImageSource(payload.data.content));

      setStatusText('正在倾听...');
    } catch (error) {
      console.error('发送语音片段失败:', error);
      setStatusText('后端请求失败，请查看控制台。');
    }
  }

  return (
    <main className="voice-canvas">
      {/* 背景光晕效果 */}
      <div className="bg-glow blob-1" aria-hidden="true" />
      <div className="bg-glow blob-2" aria-hidden="true" />

      <section className="left-column" aria-label="语音绘图控制区">
        <section className="listening-panel glass-panel" aria-label="实时语音识别">
          <div className="listening-header">
            <div className={isListening ? 'mic-orb listening' : 'mic-orb'}>
              <Mic size={26} strokeWidth={2.5} />
            </div>

            <div className={`voice-meter ${isListening ? 'active' : ''}`} aria-hidden="true">
              {waveformBars.map((bar) => (
                <span key={bar.id} style={{ animationDelay: bar.delay }} />
              ))}
            </div>
          </div>

          <div className="status-row">
            <RadioTower size={16} className={isListening ? 'text-primary' : ''} />
            <span className="status-text">{statusText}</span>
          </div>

          <div className="transcript-box">
            {speechSupported ? liveTranscript || <span className="placeholder-text">等待指令，您可以说：“画一只赛博朋克风格的猫...”</span> : '浏览器暂不支持实时语音识别'}
          </div>
        </section>

        <section className="prompt-panel glass-panel" aria-label="当前绘图提示词">
          <div className="section-title">
            <Sparkles size={16} className="text-accent" />
            <span>AI 解析指令</span>
          </div>
          <div className="prompt-box">
            {promptText || <span className="placeholder-text">识别并拆解出的绘图指令将显示在这里。</span>}
          </div>
        </section>
      </section>

      <section className="image-stage glass-panel" aria-label="绘图结果">
        {imageSrc ? (
          <img src={imageSrc} alt="AI 绘图结果" className="fade-in" />
        ) : (
          <div className="thinking-state">
            <Loader2 size={48} className="spinner text-primary" strokeWidth={1.5} />
            <div className="gradient-text">等待灵感降临</div>
            <p className="thinking-subtext">通过语音描述您想要创作的画面</p>
          </div>
        )}
      </section>
    </main>
  );
}

createRoot(document.getElementById('root')).render(<App />);
