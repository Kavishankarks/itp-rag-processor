import streamlit as st
import requests
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Configuration
API_URL = os.getenv("API_URL", "http://localhost:8002/api/v1")

st.set_page_config(
    page_title="Document Hub Search",
    page_icon="🔍",
    layout="wide"
)

def search_documents(query, search_type, limit, min_score):
    try:
        params = {
            "q": query,
            "type": search_type,
            "limit": limit,
            "min_score": min_score
        }
        response = requests.get(f"{API_URL}/search", params=params)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error connecting to API: {e}")
        return None

def generate_answer(prompt, limit, min_score, include_citations):
    try:
        payload = {
            "prompt": prompt,
            "limit": limit,
            "min_score": min_score,
            "include_citations": include_citations
        }
        response = requests.post(f"{API_URL}/generate", json=payload)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        if hasattr(e, 'response') and e.response is not None:
             try:
                 error_details = e.response.json()
                 st.error(f"API Error: {error_details.get('error', e.response.text)}")
             except:
                 st.error(f"API Error: {e.response.text}")
        else:
            st.error(f"Error connecting to API: {e}")
        return None

def upload_document(file):
    try:
        files = {"file": (file.name, file, file.type)}
        response = requests.post(f"{API_URL}/documents/upload", files=files)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error uploading document: {e}")
        return None

def convert_temp_document(file):
    try:
        files = {"file": (file.name, file, file.type)}
        response = requests.post(f"{API_URL}/convert", files=files)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error converting document: {e}")
        return None

def chunk_text(text, chunk_size, chunk_overlap):
    try:
        payload = {
            "text": text,
            "chunk_size": chunk_size,
            "chunk_overlap": chunk_overlap
        }
        response = requests.post(f"{API_URL}/chunk", json=payload)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        st.error(f"Error chunking text: {e}")
        return None

def main():
    st.title("🔍 Document Hub")
    st.markdown("Search documents, ask questions, or upload new content.")

    # Sidebar configuration
    with st.sidebar:
        st.header("Settings")
        
        # Shared settings
        limit = st.slider("Max Results / Context", min_value=1, max_value=50, value=5)
        min_score = st.slider("Min Score", min_value=0.0, max_value=1.0, value=0.3, step=0.05)
        
        st.divider()
        st.markdown(f"**API URL:** `{API_URL}`")

    # Tabs for different modes
    tab1, tab2, tab3, tab4, tab5 = st.tabs(["🔎 Search", "✨ Ask AI", "📤 Upload", "🔄 Test Convert", "✂️ Test Chunking"])

    with tab1:
        st.header("Search Documents")
        search_type = st.selectbox(
            "Search Type",
            ["hybrid", "semantic", "fulltext"],
            index=0,
            help="Hybrid combines keyword and semantic search for best results."
        )
        query = st.text_input("Enter your search query...", placeholder="e.g., 'How to configure postgres?'", key="search_query")
        
        if st.button("Search", type="primary", key="search_btn") or query:
            if not query:
                st.warning("Please enter a search query.")
            else:
                with st.spinner("Searching..."):
                    results = search_documents(query, search_type, limit, min_score)

                if results and "results" in results:
                    count = results.get("count", 0)
                    st.success(f"Found {count} results")
                    
                    for result in results["results"]:
                        # Handle potential nesting of document fields
                        doc = result.get('document', result)
                        title = doc.get('title', 'Untitled Document')
                        score = result.get('score', 0.0)
                        content = doc.get('content', '')
                        metadata = doc.get('metadata', {})
                        document_id = doc.get('id', 'Unknown')
                        
                        with st.expander(f"{title} (Score: {score:.2f})", expanded=True):
                            st.markdown(f"**Document ID:** {document_id}")
                            st.markdown("### Content Snippet")
                            st.markdown(result.get('snippet', content[:200] + "..."))
                            
                            if metadata:
                                st.markdown("### Metadata")
                                st.json(metadata)
                elif results:
                     st.info("No results found matching your criteria.")

    with tab2:
        st.header("Ask AI")
        st.markdown("Generate answers based on your documents.")
        
        prompt = st.text_area("Enter your question...", placeholder="e.g., 'Create a summary of the project architecture'", height=100)
        include_citations = st.checkbox("Include Citations", value=True)
        
        if st.button("Generate Answer", type="primary", key="gen_btn"):
            if not prompt:
                st.warning("Please enter a question.")
            else:
                with st.spinner("Generating answer... (this may take a moment)"):
                    response = generate_answer(prompt, limit, min_score, include_citations)
                
                if response:
                    st.markdown("### 🤖 Generated Answer")
                    st.markdown(response.get("generated_text", "No answer generated."))
                    
                    if response.get("sources"):
                        with st.expander("📚 Sources Used"):
                            for i, source in enumerate(response["sources"]):
                                doc = source.get('document', source)
                                st.markdown(f"**{i+1}. {doc.get('title', 'Untitled')}** (Score: {source.get('score', 0):.2f})")

    with tab3:
        st.header("Upload Document")
        st.markdown("Upload PDF, Text, or HTML files to the knowledge base.")
        
        uploaded_file = st.file_uploader("Choose a file", type=['pdf', 'txt', 'html'])
        
        if uploaded_file is not None:
            st.info(f"File selected: {uploaded_file.name} ({uploaded_file.type})")
            
            if st.button("Upload File", type="primary", key="upload_btn"):
                with st.spinner("Uploading and processing..."):
                    result = upload_document(uploaded_file)
                
                if result:
                    st.success(f"File '{uploaded_file.name}' uploaded successfully!")
                    st.json(result)

    with tab4:
        st.header("Test Conversion")
        st.markdown("Upload a file to see how it's converted to Markdown.")
        
        convert_file = st.file_uploader("Choose a file to convert", type=['pdf', 'txt', 'html'], key="convert_uploader")
        
        if convert_file is not None:
            if st.button("Convert", type="primary", key="convert_btn"):
                with st.spinner("Converting..."):
                    result = convert_temp_document(convert_file)
                
                if result:
                    st.success("Conversion successful!")
                    st.markdown("### Markdown Output")
                    st.code(result.get("markdown", ""), language="markdown")
                    
                    # Store in session state for chunking tab
                    st.session_state['last_converted_text'] = result.get("markdown", "")
                    st.info("Text preserved for the 'Test Chunking' tab.")

    with tab5:
        st.header("Test Chunking")
        st.markdown("Test the chunking logic on valid text.")
        
        # Default text from previous conversion if available
        default_text = st.session_state.get('last_converted_text', "Enter or paste text here to test chunking...")
        
        chunk_text_input = st.text_area("Text to Chunk", value=default_text, height=300, key="chunk_text_area")
        
        col1, col2 = st.columns(2)
        with col1:
            test_chunk_size = st.number_input("Chunk Size", min_value=10, value=500, step=10)
        with col2:
            test_chunk_overlap = st.number_input("Chunk Overlap", min_value=0, value=50, step=10)
            
        if st.button("Generate Chunks", type="primary", key="chunk_btn"):
            if not chunk_text_input:
                st.warning("Please enter some text to chunk.")
            else:
                with st.spinner("Chunking..."):
                    result = chunk_text(chunk_text_input, test_chunk_size, test_chunk_overlap)
                
                if result and "chunks" in result:
                    chunks = result["chunks"]
                    st.success(f"Generated {len(chunks)} chunks")
                    
                    for i, chunk in enumerate(chunks):
                        with st.expander(f"Chunk {i+1} ({len(chunk)} chars)"):
                            st.text(chunk)

if __name__ == "__main__":
    main()
