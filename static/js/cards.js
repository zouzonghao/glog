document.addEventListener('DOMContentLoaded', function() {
    const container = document.querySelector('.post-cards-container');
    if (!container) return;

    // --- Masonry Layout Logic ---
    const column1 = document.createElement('div');
    column1.className = 'card-column';
    const column2 = document.createElement('div');
    column2.className = 'card-column';
    
    const initialCards = Array.from(container.querySelectorAll('.post-card'));
    container.innerHTML = '';
    container.appendChild(column1);
    container.appendChild(column2);

    // A robust function to distribute cards based on the number of children in each column
    function distributeCards(cards) {
        cards.forEach(card => {
            if (column1.children.length <= column2.children.length) {
                column1.appendChild(card);
            } else {
                column2.appendChild(card);
            }
        });
    }

    // --- Logic for generating pseudo-covers ---
    const macaronColors = [
        '#ffb3ba', '#ffdfba', '#ffffba', '#baffc9', '#bae1ff',
        '#fec8d8', '#f2d2a9', '#f9eac3', '#c3e6cb', '#b5d8f2',
        '#f6a6b2', '#e9c39b', '#f4e0a3', '#a9d9c3', '#9cc2e5'
    ];

    function generatePseudoCover(card) {
        const cover = card.querySelector('.pseudo-cover');
        if (cover) {
            const randomColor = macaronColors[Math.floor(Math.random() * macaronColors.length)];
            cover.style.backgroundColor = randomColor;
            const title = card.dataset.title || '';
            const titleSpan = document.createElement('span');
            titleSpan.textContent = title;
            cover.innerHTML = '';
            cover.appendChild(titleSpan);
        }
    }

    // Generate covers for initial cards before distributing them
    initialCards.forEach(generatePseudoCover);
    // Distribute initial cards
    distributeCards(initialCards);

    // --- Logic for infinite scroll ---
    const trigger = document.getElementById('infinite-scroll-trigger');
    if (!trigger) return;

    const loadingIndicator = document.getElementById('loading-indicator');
    let isLoading = false;

    function loadMorePosts() {
        if (isLoading) return;

        let hasNext = container.dataset.hasNext === 'true';
        if (!hasNext) {
            loadingIndicator.innerHTML = '<p>没有更多了</p>';
            loadingIndicator.style.display = 'block';
            if (observer) observer.disconnect();
            return;
        }

        isLoading = true;
        loadingIndicator.style.display = 'block';

        const currentPage = parseInt(container.dataset.currentPage, 10);
        const nextPage = currentPage + 1;
        const url = new URL(window.location.href);
        url.searchParams.set('page', nextPage);
        
        fetch(url.toString())
            .then(response => response.text())
            .then(html => {
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                const newCards = Array.from(doc.querySelectorAll('.post-card'));
                const newContainer = doc.querySelector('.post-cards-container');

                if (newCards.length > 0) {
                    const importedCards = newCards.map(card => {
                        const importedCard = document.importNode(card, true);
                        generatePseudoCover(importedCard);
                        return importedCard;
                    });
                    // Distribute newly loaded cards
                    distributeCards(importedCards);
                }

                if (newContainer) {
                    container.dataset.currentPage = newContainer.dataset.currentPage;
                    container.dataset.hasNext = newContainer.dataset.hasNext;
                } else {
                    container.dataset.hasNext = 'false';
                }
                
                isLoading = false;
                loadingIndicator.style.display = 'none';

                if (container.dataset.hasNext === 'false') {
                    loadingIndicator.innerHTML = '<p>没有更多了</p>';
                    loadingIndicator.style.display = 'block';
                    if (observer) observer.disconnect();
                }
            })
            .catch(error => {
                console.error('Error loading more posts:', error);
                loadingIndicator.innerHTML = '<p>加载失败，请重试</p>';
                isLoading = false;
            });
    }

    const observer = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting) {
            loadMorePosts();
        }
    }, {
        rootMargin: '200px'
    });

    observer.observe(trigger);
});