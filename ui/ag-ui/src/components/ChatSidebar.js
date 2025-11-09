import React, { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import './ChatSidebar.css';

function ChatSidebar({ onQuery, response }) {
  const [query, setQuery] = useState('');
  const [chatHistory, setChatHistory] = useState([]);
  const [pendingQuery, setPendingQuery] = useState(null);
  const messagesEndRef = useRef(null);
  const lastResponseRef = useRef('');

  // Scroll to bottom when new messages are added
  const scrollToBottom = () => {
    if (messagesEndRef.current) {
      const messagesContainer = messagesEndRef.current.parentElement;
      if (messagesContainer) {
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
      }
    }
  };

  useEffect(() => {
    // Use setTimeout to ensure DOM has updated
    const timeoutId = setTimeout(scrollToBottom, 0);
    return () => clearTimeout(timeoutId);
  }, [chatHistory]);

  // Handle incoming response
  useEffect(() => {
    if (response && response !== lastResponseRef.current && response.trim() !== '') {
      // Check if this response is already in chat history
      const alreadyAdded = chatHistory.some(
        msg => msg.type === 'assistant' && msg.text === response
      );
      
      if (!alreadyAdded) {
        // Add the assistant response to chat history
        setChatHistory(prev => [...prev, { type: 'assistant', text: response }]);
        lastResponseRef.current = response;
        setPendingQuery(null);
      }
    }
  }, [response, chatHistory]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!query.trim()) return;

    const userQuery = query.trim();
    const userMessage = { type: 'user', text: userQuery };
    
    // Add user message immediately
    setChatHistory(prev => [...prev, userMessage]);
    setPendingQuery(userQuery);
    
    // Clear input immediately
    setQuery('');
    
    // Call the query handler (async) with error handling
    try {
      await onQuery(userQuery);
    } catch (error) {
      console.error('Error in handleSubmit:', error);
      // Add error message to chat
      setChatHistory(prev => [...prev, { 
        type: 'assistant', 
        text: `❌ Failed to process query: ${error.message}` 
      }]);
      setPendingQuery(null);
    }
  };

  return (
    <div className="chat-sidebar">
      <div className="chat-header">
        <h2>StackMonitor AI Assistant</h2>
      </div>
      <div className="chat-messages">
        {chatHistory.length === 0 ? (
          <div className="welcome-message">
            <p>Ask me about your logs and metrics!</p>
            <p className="hint">Try: "Show me recent errors" or "What's the error rate?"</p>
          </div>
        ) : (
          <>
            {chatHistory.map((msg, idx) => (
              <div key={`${msg.type}-${idx}-${msg.text.substring(0, 20)}`} className={`chat-message ${msg.type}`}>
                <div className="message-content">
                  <ReactMarkdown 
                    remarkPlugins={[remarkGfm]}
                    components={{
                      a: ({node, ...props}) => (
                        <a 
                          {...props} 
                          target="_blank" 
                          rel="noopener noreferrer"
                          style={{
                            color: '#2196f3',
                            textDecoration: 'underline',
                            fontWeight: '500',
                            cursor: 'pointer'
                          }}
                        />
                      )
                    }}
                  >
                    {msg.text}
                  </ReactMarkdown>
                </div>
              </div>
            ))}
            {pendingQuery && !chatHistory.some(m => m.type === 'assistant' && m.text === response) && (
              <div className="chat-message assistant loading">
                <div className="message-content">
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>
                    Thinking...
                  </ReactMarkdown>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>
      <form className="chat-input-form" onSubmit={handleSubmit}>
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Ask about logs or metrics..."
          className="chat-input"
        />
        <button type="submit" className="chat-send-button">Send</button>
      </form>
    </div>
  );
}

export default ChatSidebar;
