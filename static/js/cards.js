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
    
    function handleCoverError(img) {
        const cardCover = img.parentElement;
        const card = cardCover.closest('.post-card');
        cardCover.innerHTML = '<div class="pseudo-cover"></div>';
        generatePseudoCover(card);
    }
}

document.addEventListener('DOMContentLoaded', function() {
    const container = document.querySelector('.post-cards-container');
    if (!container) return;

    const column1 = document.createElement('div');
    column1.className = 'card-column';
    const column2 = document.createElement('div');
    column2.className = 'card-column';
    
    const initialCards = Array.from(container.querySelectorAll('.post-card'));
    container.innerHTML = '';
    container.appendChild(column1);
    container.appendChild(column2);

    function distributeCards(cards) {
        const frag1 = document.createDocumentFragment();
        const frag2 = document.createDocumentFragment();
        let count1 = column1.children.length;
        let count2 = column2.children.length;

        cards.forEach(card => {
            if (count1 <= count2) {
                frag1.appendChild(card);
                count1++;
            } else {
                frag2.appendChild(card);
                count2++;
            }
        });

        if (frag1.childElementCount > 0) column1.appendChild(frag1);
        if (frag2.childElementCount > 0) column2.appendChild(frag2);
    }

    initialCards.forEach(generatePseudoCover);
    
    requestAnimationFrame(() => {
        distributeCards(initialCards);
        container.style.opacity = '1';
    });

    container.addEventListener('error', (event) => {
        if (event.target.tagName === 'IMG') {
            const cardCover = event.target.parentElement;
            const card = cardCover.closest('.post-card');
            cardCover.innerHTML = '<div class="pseudo-cover"></div>';
            generatePseudoCover(card);
        }
    }, true);

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
