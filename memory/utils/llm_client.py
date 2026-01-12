"""
LLM Client - Handles all LLM interactions
Configured for OpenRouter free tier with automatic fallback
"""
import json
from typing import List, Dict, Any, Optional
from openai import OpenAI
import config


class LLMClient:
    """
    Unified LLM client interface with free tier support, fallback models,
    and multiple API key rotation for rate limit resilience.
    """
    def __init__(
        self,
        api_key: Optional[str] = None,
        model: Optional[str] = None,
        base_url: Optional[str] = None,
        enable_thinking: Optional[bool] = None,
        use_streaming: Optional[bool] = None
    ):
        self.model = model or config.LLM_MODEL
        self.base_url = base_url or config.OPENAI_BASE_URL
        self.enable_thinking = enable_thinking if enable_thinking is not None else config.ENABLE_THINKING
        self.use_streaming = use_streaming if use_streaming is not None else config.USE_STREAMING
        
        # API keys for rotation (primary + backup)
        if api_key:
            self.api_keys = [api_key]
        elif hasattr(config, 'get_api_keys'):
            self.api_keys = config.get_api_keys()
        else:
            self.api_keys = [config.OPENAI_API_KEY]
        
        self.current_key_index = 0
        self.api_key = self.api_keys[0]
        
        # Fallback models for free tier rotation
        self.fallback_models = getattr(config, 'LLM_FALLBACK_MODELS', [])
        self.current_model_index = 0
        
        # OpenRouter headers
        self.default_headers = getattr(config, 'OPENROUTER_HEADERS', {})

        # Initialize client with primary key
        self.client = self._create_client(self.api_key)
        
        # Log configuration
        print(f"Using OpenRouter: {self.base_url}")
        print(f"API keys available: {len(self.api_keys)} (primary{' + backup' if len(self.api_keys) > 1 else ''})")
        print(f"Default model: {self.model} (FREE TIER)")
        if self.default_headers:
            print(f"OpenRouter headers configured")
        if self.enable_thinking:
            print(f"Deep thinking mode enabled")

    def _create_client(self, api_key: str) -> OpenAI:
        """Create OpenAI client with given API key."""
        client_kwargs = {"api_key": api_key}
        if self.base_url:
            client_kwargs["base_url"] = self.base_url
        if self.default_headers:
            client_kwargs["default_headers"] = self.default_headers
        return OpenAI(**client_kwargs)
    
    def _rotate_api_key(self) -> bool:
        """
        Rotate to the next available API key.
        Returns True if rotation successful, False if no more keys.
        """
        if len(self.api_keys) <= 1:
            return False
        
        self.current_key_index = (self.current_key_index + 1) % len(self.api_keys)
        
        # Check if we've cycled through all keys
        if self.current_key_index == 0:
            return False  # All keys exhausted
        
        self.api_key = self.api_keys[self.current_key_index]
        self.client = self._create_client(self.api_key)
        print(f"Rotated to API key #{self.current_key_index + 1}")
        return True

    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        temperature: float = 0.2,
        response_format: Optional[Dict[str, str]] = None,
        max_retries: int = 3
    ) -> str:
        """
        Standard chat completion with optional thinking mode and retry mechanism
        """
        kwargs = {
            "model": self.model,
            "messages": messages,
            "temperature": temperature,
        }

        if response_format:
            kwargs["response_format"] = response_format

        # Enable thinking mode if configured (for Qwen and compatible models only)
        # Only add enable_thinking parameter for Qwen API (identified by base_url)
        is_qwen_api = self.base_url and "dashscope.aliyuncs.com" in self.base_url
        
        if is_qwen_api:
            # Qwen API requires explicit enable_thinking parameter
            # - Streaming + thinking: enable_thinking=True
            # - Non-streaming: enable_thinking=False (required, not optional)
            # - JSON format: enable_thinking=False (incompatible with thinking mode)
            if self.use_streaming and self.enable_thinking and not response_format:
                kwargs["extra_body"] = {"enable_thinking": True}
            else:
                # Explicitly set to False for non-streaming calls or JSON format
                kwargs["extra_body"] = {"enable_thinking": False}
        # For OpenAI and other APIs, don't add extra_body parameters

        # Retry mechanism with API key rotation and model fallback
        last_exception = None
        models_to_try = [self.model] + self.fallback_models
        
        # Track which API keys we've tried
        keys_tried = 0
        max_key_rotations = len(self.api_keys)
        
        while keys_tried < max_key_rotations:
            for model in models_to_try:
                kwargs["model"] = model
                
                for attempt in range(max_retries):
                    try:
                        # Use streaming if configured
                        if self.use_streaming:
                            kwargs["stream"] = True
                            return self._handle_streaming_response(**kwargs)
                        else:
                            response = self.client.chat.completions.create(**kwargs)
                            return response.choices[0].message.content
                            
                    except Exception as e:
                        last_exception = e
                        error_str = str(e).lower()
                        
                        # Check if rate limited - try rotating API key first
                        if "rate limit" in error_str or "429" in error_str:
                            print(f"Rate limited on key #{self.current_key_index + 1}, model {model}")
                            
                            # Try rotating API key before switching models
                            if self._rotate_api_key():
                                print(f"Switched to backup API key, retrying...")
                                keys_tried += 1
                                break  # Retry with new key, same model
                            else:
                                print(f"No more API keys, trying fallback model...")
                                break  # Try next model
                        
                        # Model unavailable - try fallback model
                        if "unavailable" in error_str or "not found" in error_str:
                            print(f"Model {model} unavailable, trying fallback...")
                            break  # Try next model
                        
                        if attempt < max_retries - 1:
                            import time
                            wait_time = (2 ** attempt)  # Exponential backoff: 1s, 2s, 4s
                            print(f"LLM API call failed (attempt {attempt + 1}/{max_retries}): {e}")
                            print(f"Retrying in {wait_time} seconds...")
                            time.sleep(wait_time)
                        else:
                            print(f"Model {model} failed after {max_retries} attempts, trying fallback...")
                            break  # Try next model
                else:
                    # Inner loop completed without break = success would have returned
                    continue
                break  # Break out to try key rotation or next model
            else:
                # All models tried with current key
                if self._rotate_api_key():
                    keys_tried += 1
                    continue  # Try all models with new key
                else:
                    break  # No more keys
            
            # If we get here from a break, we need to continue the while loop
            continue
        
        # If all keys and models failed, raise the last exception
        print(f"All API keys and models exhausted. Last error: {last_exception}")
        raise last_exception

    def _handle_streaming_response(self, **kwargs) -> str:
        """
        Handle streaming response and collect full content
        """
        full_content = []
        stream = self.client.chat.completions.create(**kwargs)

        for chunk in stream:
            if chunk.choices[0].delta.content is not None:
                content = chunk.choices[0].delta.content
                full_content.append(content)
                # Optional: print streaming content in real-time
                # print(content, end='', flush=True)

        return ''.join(full_content)

    def extract_json(self, text: str) -> Any:
        """
        Extract JSON from LLM response with robust parsing
        Supports multiple formats:
        1. Pure JSON
        2. ```json ... ```
        3. ``` ... ``` (generic code block)
        4. JSON embedded in text with common prefixes
        5. Multiple JSON objects (returns first valid one)
        """
        if not text or not text.strip():
            raise ValueError("Empty response received")

        text = text.strip()

        # Remove common LLM prefixes/suffixes
        common_prefixes = [
            "Here's the JSON:",
            "Here is the JSON:",
            "The JSON is:",
            "JSON:",
            "Result:",
            "Output:",
            "Answer:",
        ]
        for prefix in common_prefixes:
            if text.lower().startswith(prefix.lower()):
                text = text[len(prefix):].strip()

        # Try direct parsing first
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            pass

        # Try extracting JSON from ```json ... ``` block
        if "```json" in text.lower():
            # Case insensitive search for ```json
            start_marker = "```json"
            start_idx = text.lower().find(start_marker)
            if start_idx != -1:
                start = start_idx + len(start_marker)
                # Find the closing ```
                end = text.find("```", start)
                if end != -1:
                    json_str = text[start:end].strip()
                    try:
                        return json.loads(json_str)
                    except json.JSONDecodeError as e:
                        # Try to clean up common issues
                        json_str = self._clean_json_string(json_str)
                        try:
                            return json.loads(json_str)
                        except json.JSONDecodeError:
                            pass

        # Try extracting from generic ``` ... ``` code block
        if "```" in text:
            start = text.find("```") + 3
            # Skip language identifier if present
            newline = text.find("\n", start)
            if newline != -1 and newline - start < 20:
                start = newline + 1
            end = text.find("```", start)
            if end != -1:
                json_str = text[start:end].strip()
                try:
                    return json.loads(json_str)
                except json.JSONDecodeError:
                    # Try to clean up
                    json_str = self._clean_json_string(json_str)
                    try:
                        return json.loads(json_str)
                    except json.JSONDecodeError:
                        pass

        # Try finding balanced JSON object/array by scanning for { or [
        for start_char in ['{', '[']:
            result = self._extract_balanced_json(text, start_char)
            if result is not None:
                return result

        # Last resort: try to find any JSON-like structure and clean it
        for start_char in ['{', '[']:
            start_idx = text.find(start_char)
            if start_idx != -1:
                # Extract a large chunk and try to parse
                chunk = text[start_idx:]
                cleaned = self._clean_json_string(chunk)
                try:
                    return json.loads(cleaned)
                except json.JSONDecodeError:
                    pass

        raise ValueError(f"Failed to extract valid JSON from response. First 300 chars: {text[:300]}...")

    def _clean_json_string(self, json_str: str) -> str:
        """
        Clean common issues in JSON strings from LLM output
        """
        # Remove trailing commas before } or ]
        import re
        json_str = re.sub(r',(\s*[}\]])', r'\1', json_str)

        # Remove comments (// and /* */)
        json_str = re.sub(r'//.*?$', '', json_str, flags=re.MULTILINE)
        json_str = re.sub(r'/\*.*?\*/', '', json_str, flags=re.DOTALL)

        return json_str.strip()

    def _extract_balanced_json(self, text: str, start_char: str) -> Any:
        """
        Extract a balanced JSON object or array starting with start_char
        """
        end_char = '}' if start_char == '{' else ']'
        start_idx = text.find(start_char)

        if start_idx == -1:
            return None

        # Track depth to find matching closing bracket
        depth = 0
        in_string = False
        escape_next = False

        for i in range(start_idx, len(text)):
            char = text[i]

            # Handle string escaping
            if escape_next:
                escape_next = False
                continue

            if char == '\\':
                escape_next = True
                continue

            # Handle strings (don't count brackets inside strings)
            if char == '"':
                in_string = not in_string
                continue

            if in_string:
                continue

            # Count depth
            if char == start_char:
                depth += 1
            elif char == end_char:
                depth -= 1
                if depth == 0:
                    json_str = text[start_idx:i+1]
                    try:
                        return json.loads(json_str)
                    except json.JSONDecodeError:
                        # Try cleaning and parsing again
                        cleaned = self._clean_json_string(json_str)
                        try:
                            return json.loads(cleaned)
                        except json.JSONDecodeError:
                            # Continue searching for next occurrence
                            break

        return None
