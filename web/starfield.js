document.addEventListener('DOMContentLoaded', () => {
    const canvas = document.getElementById('starfield');
    const ctx = canvas.getContext('2d');
    const container = document.getElementById('starfield-container');
    
    // Set canvas size to match window
    function resizeCanvas() {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    }
    
    window.addEventListener('resize', resizeCanvas);
    resizeCanvas();
    
    // Star class
    class Star {
        constructor() {
            this.reset();
            this.z = Math.random() * 2000;
        }
        
        reset() {
            this.x = Math.random() * canvas.width - canvas.width / 2;
            this.y = Math.random() * canvas.height - canvas.height / 2;
            this.size = Math.random() * 2 + 0.5;
            this.baseSize = this.size;
            this.speed = Math.random() * 0.02 + 0.005;
            this.color = `rgba(255, 255, 255, ${Math.random() * 0.5 + 0.5})`;
            this.z = Math.random() * 2000;
        }
        
        update(mouseX, mouseY, mouseInfluence) {
            // Move star away from mouse position
            if (mouseInfluence > 0) {
                const dx = this.x - mouseX;
                const dy = this.y - mouseY;
                const distance = Math.sqrt(dx * dx + dy * dy);
                
                if (distance < 200) {
                    const force = (200 - distance) / 200;
                    this.x += dx * force * 0.1;
                    this.y += dy * force * 0.1;
                }
            }
            
            // Move star towards viewer (create depth effect)
            this.z -= 1;
            
            // Reset star if it passes the viewer
            if (this.z <= 0) {
                this.reset();
                this.z = 2000;
            }
            
            // Pulsing effect
            this.size = this.baseSize + Math.sin(Date.now() * this.speed) * 0.5;
        }
        
        draw() {
            const x = (this.x / this.z) * canvas.width/2 + canvas.width/2;
            const y = (this.y / this.z) * canvas.height/2 + canvas.height/2;
            const size = (this.size / this.z) * 1000;
            
            if (x > 0 && x < canvas.width && y > 0 && y < canvas.height && size > 0) {
                ctx.beginPath();
                ctx.arc(x, y, size, 0, Math.PI * 2);
                ctx.fillStyle = this.color;
                ctx.fill();
            }
        }
    }
    
    // Create stars
    const stars = [];
    const starCount = 1000;
    
    for (let i = 0; i < starCount; i++) {
        stars.push(new Star());
    }
    
    // Mouse position tracking
    let mouseX = 0;
    let mouseY = 0;
    let mouseInfluence = 0;
    
    container.addEventListener('mousemove', (e) => {
        mouseX = e.clientX - canvas.width / 2;
        mouseY = e.clientY - canvas.height / 2;
        mouseInfluence = 1;
        
        // Hide instructions after first interaction
        const instructions = document.getElementById('instructions');
        instructions.style.opacity = '0';
    });
    
    container.addEventListener('mouseleave', () => {
        mouseInfluence = 0;
    });
    
    // Animation loop
    function animate() {
        // Clear canvas with a semi-transparent black for trail effect
        ctx.fillStyle = 'rgba(0, 0, 0, 0.1)';
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        // Update and draw stars
        stars.forEach(star => {
            star.update(mouseX, mouseY, mouseInfluence);
            star.draw();
        });
        
        requestAnimationFrame(animate);
    }
    
    // Start animation
    animate();
});