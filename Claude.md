Documentation Hub v1 - Plan                                                                                                    
                                                                                                                                    
     Overview                                                                                                                       
                                                                                                                                    
     Build a Python application that saves documentation to PostgreSQL and provides full-text + semantic vector search capabilities.
                                                                                                                                    
     Database Setup                                                                                                                 
                                                                                                                                    
     1. Install pgvector extension (for semantic search)                                                                            
     2. Create new database: doc_hub                                                                                                
     3. Enable extensions: pgvector and pg_trgm                                                                                     
     4. Schema design:                                                                                                              
       - documents table: id, title, content, source_url, doc_type, metadata (JSON), created_at, updated_at                         
       - document_chunks table: id, doc_id, chunk_text, chunk_index, embedding (vector[768]), created_at                            
       - Indexes: GiST for full-text, IVFFlat/HNSW for vector similarity                                                            
                                                                                                                                    
     Application Architecture                                                                                                       
                                                                                                                                    
     Tech Stack:                                                                                                                    
     - FastAPI (REST API framework)                                                                                                 
     - SQLAlchemy (ORM)                                                                                                             
     - Alembic (migrations)                                                                                                         
     - sentence-transformers (embeddings - all-MiniLM-L6-v2 model)                                                                  
     - psycopg2 (PostgreSQL driver)                                                                                                 
                                                                                                                                    
     Project Structure:                                                                                                             
     itp-rag-processor/                                                                                                                  
     ├── app/                                                                                                                       
       ├── main.py          # FastAPI app & routes                                                                                
       ├── models.py        # SQLAlchemy models                                                                                   
       ├── schemas.py       # Pydantic request/response models                                                                    
       ├── database.py      # DB connection & session                                                                             
       ├── crud.py          # Database operations                                                                                 
       ├── search.py        # Search logic (full-text + vector)                                                                   
       └── embeddings.py    # Embedding generation                                                                                
     ├── alembic/             # DB migrations                                                                                       
     ├── tests/                                                                                                                     
     ├── requirements.txt                                                                                                           
     ├── .env                                                                                                                       
     └── README.md                                                                                                                  
                                                                                                                                    
     v1 Features                                                                                                                    
                                                                                                                                    
     1. Document Management:                                                                                                        
     - Save documentation (accepts text/markdown)                                                                                   
     - Auto-chunk large docs (~500 tokens per chunk)                                                                                
     - Generate embeddings for each chunk                                                                                           
     - Store metadata (source, type, tags)                                                                                          
     - Update & delete operations                                                                                                   
                                                                                                                                    
     2. Search:                                                                                                                     
     - Full-text search: PostgreSQL pg_trgm + ts_vector for keyword matching                                                        
     - Semantic search: pgvector cosine similarity for meaning-based search                                                         
     - Hybrid search: Combine both with weighted scoring (0.4 full-text + 0.6 semantic)                                             
     - Return ranked results with snippets                                                                                          
                                                                                                                                    
     3. API Endpoints:                                                                                                              
     - POST /documents - Upload documentation                                                                                       
     - GET /documents/{id} - Retrieve document                                                                                      
     - PUT /documents/{id} - Update document                                                                                        
     - DELETE /documents/{id} - Delete document                                                                                     
     - GET /search?q=query&type=[fulltext|semantic|hybrid]&limit=10 - Search                                                        
     - GET /documents?skip=0&limit=20 - List documents                                                                              
                                                                                                                                    
     Implementation Steps                                                                                                           
                                                                                                                                    
     1. Set up project structure & dependencies                                                                                     
     2. Install pgvector and create database                                                                                        
     3. Define SQLAlchemy models & migrations                                                                                       
     4. Implement embedding generation service                                                                                      
     5. Create CRUD operations for documents                                                                                        
     6. Build search functionality (all 3 types)                                                                                    
     7. Create FastAPI endpoints                                                                                                    
     8. Add basic tests                                                                                                             
     9. Create README with usage examples 