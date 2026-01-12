"""
Ollama Embedding Model - Local embedding using Ollama
Replaces SentenceTransformers for local-first embedding
"""
from typing import List, Optional
import numpy as np
import requests
import config


class OllamaEmbeddingModel:
    """
    Embedding model using Ollama's local embedding API
    """
    def __init__(self, model_name: str = None, base_url: str = None):
        self.model_name = model_name or config.EMBEDDING_MODEL
        self.base_url = base_url or config.OLLAMA_BASE_URL
        self.dimension = config.EMBEDDING_DIMENSION
        self.model_type = "ollama"
        self.supports_query_prompt = False
        
        # Strip "ollama:" prefix if present
        if self.model_name.startswith("ollama:"):
            self.model_name = self.model_name[7:]
        
        print(f"Initializing Ollama embedding model: {self.model_name}")
        print(f"Ollama endpoint: {self.base_url}")
        
        # Verify connection
        self._verify_connection()

    def _verify_connection(self):
        """Verify Ollama is running and model is available"""
        try:
            response = requests.get(f"{self.base_url}/api/tags", timeout=5)
            if response.status_code == 200:
                models = response.json().get("models", [])
                model_names = [m.get("name", "").split(":")[0] for m in models]
                if self.model_name not in model_names and f"{self.model_name}:latest" not in [m.get("name") for m in models]:
                    print(f"Warning: Model '{self.model_name}' not found. Available: {model_names}")
                    print(f"Run: ollama pull {self.model_name}")
                else:
                    print(f"Ollama model '{self.model_name}' verified")
            else:
                print(f"Warning: Could not verify Ollama models (status {response.status_code})")
        except requests.exceptions.ConnectionError:
            print(f"Warning: Ollama not running at {self.base_url}")
            print("Start Ollama with: ollama serve")
        except Exception as e:
            print(f"Warning: Ollama verification failed: {e}")

    def _get_embedding(self, text: str) -> np.ndarray:
        """Get embedding for a single text from Ollama"""
        try:
            response = requests.post(
                f"{self.base_url}/api/embeddings",
                json={
                    "model": self.model_name,
                    "prompt": text
                },
                timeout=30
            )
            
            if response.status_code == 200:
                embedding = response.json().get("embedding", [])
                if embedding:
                    return np.array(embedding, dtype=np.float32)
                else:
                    raise ValueError("Empty embedding returned")
            else:
                raise ValueError(f"Ollama API error: {response.status_code} - {response.text}")
                
        except requests.exceptions.ConnectionError:
            raise ConnectionError(f"Cannot connect to Ollama at {self.base_url}. Run: ollama serve")
        except Exception as e:
            raise RuntimeError(f"Embedding failed: {e}")

    def encode(self, texts: List[str], is_query: bool = False) -> np.ndarray:
        """
        Encode list of texts to vectors
        
        Args:
        - texts: List of texts to encode
        - is_query: Whether these are query texts (not used for Ollama, but kept for API compatibility)
        """
        if isinstance(texts, str):
            texts = [texts]
        
        embeddings = []
        for text in texts:
            embedding = self._get_embedding(text)
            # Normalize
            norm = np.linalg.norm(embedding)
            if norm > 0:
                embedding = embedding / norm
            embeddings.append(embedding)
        
        return np.array(embeddings, dtype=np.float32)

    def encode_single(self, text: str, is_query: bool = False) -> np.ndarray:
        """Encode single text"""
        return self.encode([text], is_query=is_query)[0]
    
    def encode_query(self, queries: List[str]) -> np.ndarray:
        """Encode queries"""
        return self.encode(queries, is_query=True)
    
    def encode_documents(self, documents: List[str]) -> np.ndarray:
        """Encode documents"""
        return self.encode(documents, is_query=False)


class EmbeddingModel:
    """
    Factory class that returns appropriate embedding model based on config
    """
    def __new__(cls, model_name: str = None, use_optimization: bool = True):
        model_name = model_name or config.EMBEDDING_MODEL
        
        # Use Ollama if model starts with "ollama:"
        if model_name.startswith("ollama:"):
            return OllamaEmbeddingModel(model_name)
        
        # Otherwise, use original SentenceTransformers implementation
        from utils.embedding_original import EmbeddingModel as OriginalEmbeddingModel
        return OriginalEmbeddingModel(model_name, use_optimization)
