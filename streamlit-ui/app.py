import streamlit as st
import requests
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Configuration
API_URL = os.getenv("API_URL", "http://localhost:8000/api/v1")

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

def main():
    st.title("🔍 Document Hub")
    st.markdown("Search documents or ask questions using AI.")

    # Sidebar configuration
    with st.sidebar:
        st.header("Settings")
        
        # Shared settings
        limit = st.slider("Max Results / Context", min_value=1, max_value=50, value=5)
        min_score = st.slider("Min Score", min_value=0.0, max_value=1.0, value=0.3, step=0.05)
        
        st.divider()
        st.markdown(f"**API URL:** `{API_URL}`")

    # Tabs for different modes
    tab1, tab2 = st.tabs(["🔎 Search", "✨ Ask AI"])

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

if __name__ == "__main__":
    main()
