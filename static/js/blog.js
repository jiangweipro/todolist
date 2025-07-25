document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const blogTitle = document.getElementById('blog-title');
    const blogAuthor = document.getElementById('blog-author');
    const blogDate = document.getElementById('blog-date');
    const blogContent = document.getElementById('blog-content');
    const privateBadge = document.getElementById('private-badge');
    const commentsList = document.getElementById('comments-list');
    const commentTemplate = document.getElementById('comment-item-template');
    const commentInput = document.getElementById('comment-input');
    const commentSubmit = document.getElementById('comment-submit');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    const backBtn = document.getElementById('back-btn');
    const editBtn = document.getElementById('edit-btn');

    // 获取当前用户信息
    getCurrentUser();
    
    // 获取博客ID
    const blogId = window.location.pathname.split('/').pop();
    
    // 加载博客详情
    loadBlog(blogId);
    
    // 登出按钮事件监听
    if (logoutBtn) {
        logoutBtn.addEventListener('click', logout);
    }
    
    // 返回按钮事件监听
    if (backBtn) {
        backBtn.addEventListener('click', () => {
            window.location.href = '/blogs';
        });
    }
    
    // 评论提交按钮事件监听
    if (commentSubmit) {
        commentSubmit.addEventListener('click', () => {
            addComment(blogId);
        });
    }

    // 当前用户信息
    let currentUser = null;
    let currentBlog = null;
    
    // 获取当前用户信息
    async function getCurrentUser() {
        try {
            const response = await fetch('/api/current-user');
            if (response.ok) {
                const userData = await response.json();
                // 保存当前用户信息
                currentUser = userData;
                
                if (usernameElement) {
                    usernameElement.textContent = userData.username;
                }
                
                // 显示管理员标识
                const adminBadge = document.getElementById('admin-badge');
                if (adminBadge && userData.is_admin) {
                    adminBadge.style.display = 'inline-block';
                }
            } else {
                // 如果未登录，重定向到登录页面
                window.location.href = '/login';
            }
        } catch (error) {
            console.error('获取用户信息失败:', error);
            window.location.href = '/login';
        }
    }
    
    // 登出功能
    async function logout() {
        try {
            const response = await fetch('/logout', {
                method: 'POST'
            });
            
            if (response.ok) {
                window.location.href = '/login';
            }
        } catch (error) {
            console.error('登出失败:', error);
        }
    }
    
    // 加载博客详情
    async function loadBlog(id) {
        try {
            const response = await fetch(`/api/blogs/${id}`);
            if (!response.ok) {
                // 如果博客不存在或无权访问，返回博客列表页
                window.location.href = '/blogs';
                return;
            }
            
            const blog = await response.json();
            currentBlog = blog;
            
            // 设置博客详情
            blogTitle.textContent = blog.title;
            blogAuthor.textContent = blog.username || `用户 ${blog.user_id}`;
            blogDate.textContent = new Date(blog.created_at).toLocaleString();
            blogContent.textContent = blog.content;
            
            // 如果是私密博客，显示私密标识
            if (blog.is_private) {
                privateBadge.style.display = 'inline-block';
            }
            
            // 如果是当前用户的博客，显示编辑按钮
            if (currentUser && blog.user_id === currentUser.id) {
                editBtn.style.display = 'inline-block';
                editBtn.addEventListener('click', () => {
                    window.location.href = `/blogs/edit/${blog.id}`;
                });
            }
            
            // 加载评论
            loadComments(blog.comments);
        } catch (error) {
            console.error('加载博客失败:', error);
            window.location.href = '/blogs';
        }
    }
    
    // 加载评论
    function loadComments(comments) {
        // 清空评论列表
        commentsList.innerHTML = '';
        
        // 添加所有评论到列表
        comments.forEach(comment => {
            appendCommentToDOM(comment);
        });
    }
    
    // 添加评论
    async function addComment(blogId) {
        const content = commentInput.value.trim();
        if (!content) return;
        
        try {
            const response = await fetch(`/api/blogs/comments/${blogId}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ content })
            });
            
            if (response.ok) {
                const comment = await response.json();
                appendCommentToDOM(comment);
                commentInput.value = '';
            }
        } catch (error) {
            console.error('添加评论失败:', error);
        }
    }
    
    // 删除评论
    async function deleteComment(commentId, commentElement) {
        try {
            // 获取博客ID
            const blogId = window.location.pathname.split('/').pop();
            // 修正API路径格式
            const response = await fetch(`/api/blogs/comments/${blogId}/${commentId}`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                commentElement.remove();
            } else {
                // 如果删除失败，显示错误信息
                const errorData = await response.json().catch(() => ({ message: '删除评论失败' }));
                alert(errorData.message || '删除评论失败');
            }
        } catch (error) {
            console.error('删除评论失败:', error);
            alert('删除评论失败: ' + error.message);
        }
    }
    
    // 将评论添加到DOM
    function appendCommentToDOM(comment) {
        // 克隆模板
        const commentNode = document.importNode(commentTemplate.content, true);
        const commentItem = commentNode.querySelector('.comment-item');
        const commentAuthor = commentNode.querySelector('.comment-author');
        const commentDate = commentNode.querySelector('.comment-date');
        const commentContent = commentNode.querySelector('.comment-content');
        const deleteCommentBtn = commentNode.querySelector('.delete-comment-btn');
        
        // 设置数据
        commentItem.dataset.id = comment.id;
        commentItem.dataset.userId = comment.user_id;
        commentAuthor.textContent = comment.username || `用户 ${comment.user_id}`;
        commentDate.textContent = new Date(comment.created_at).toLocaleString();
        commentContent.textContent = comment.content;
        
        // 如果是当前用户的评论或当前用户是博客作者，显示删除按钮
        if (currentUser && (comment.user_id === currentUser.id || (currentBlog && currentBlog.user_id === currentUser.id))) {
            deleteCommentBtn.style.display = 'inline-block';
            deleteCommentBtn.addEventListener('click', () => {
                if (confirm('确定要删除这条评论吗？')) {
                    deleteComment(comment.id, commentItem);
                }
            });
        }
        
        // 添加到列表
        commentsList.appendChild(commentNode);
    }
});