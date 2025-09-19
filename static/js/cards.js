document.addEventListener('DOMContentLoaded', function() {
    // Select only the containers that need a pseudo-image
    const pseudoCovers = document.querySelectorAll('.pseudo-cover');
    
    // A beautiful palette of macaron colors
    const macaronColors = [
        '#ffb3ba', '#ffdfba', '#ffffba', '#baffc9', '#bae1ff',
        '#fec8d8', '#f2d2a9', '#f9eac3', '#c3e6cb', '#b5d8f2',
        '#f6a6b2', '#e9c39b', '#f4e0a3', '#a9d9c3', '#9cc2e5'
    ];

    pseudoCovers.forEach(cover => {
        // Find the parent .post-card to get the title
        const card = cover.closest('.post-card');
        if (card) {
            // Select a random color from the palette
            const randomColor = macaronColors[Math.floor(Math.random() * macaronColors.length)];
            cover.style.backgroundColor = randomColor;

            // Get the title from the data attribute of the parent card
            const title = card.dataset.title || '';

            // Create a span to hold the title text inside the cover
            const titleSpan = document.createElement('span');
            titleSpan.textContent = title;
            
            // Clear any existing content and append the new title span
            cover.innerHTML = ''; 
            cover.appendChild(titleSpan);
        }
    });
});