document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const blogList = document.getElementById('blog-list');
    const blogTemplate = document.getElementById('blog-item-template');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    const backBtn = document.getElementById('back-btn');
    const newBlogBtn = document.getElementById('new-blog-btn');

    // 获取当前用户信息
    getCurrentUser();
    
    // 加载所有博客
    loadBlogs();
    
    // 登出按钮事件监听
    if (logoutBtn) {
        logoutBtn.addEventListener('click', logout);
    }
    
    // 返回按钮事件监听
    if (backBtn) {
        backBtn.addEventListener('click', () => {
            window.location.href = '/';
        });
    }
    
    // 新建博客按钮事件监听
    if (newBlogBtn) {
        newBlogBtn.addEventListener('click', () => {
            window.location.href = '/blogs/new';
        });
    }

    // 当前用户信息
    let currentUser = null;
    
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
    
    // 加载所有博客
    async function loadBlogs() {
        try {
            const response = await fetch('/api/blogs');
            const blogs = await response.json();
            
            // 清空列表
            blogList.innerHTML = '';
            
            // 添加所有博客到列表
            blogs.forEach(blog => {
                appendBlogToDOM(blog);
            });
        } catch (error) {
            console.error('加载博客失败:', error);
        }
    }

    // 删除博客
    async function deleteBlog(id, blogElement) {
        try {
            const response = await fetch(`/api/blogs/${id}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                blogElement.remove();
            }
        } catch (error) {
            console.error('删除博客失败:', error);
        }
    }

    // 将博客添加到DOM
    function appendBlogToDOM(blog) {
        // 克隆模板
        const blogNode = document.importNode(blogTemplate.content, true);
        const blogItem = blogNode.querySelector('.blog-item');
        const blogTitle = blogNode.querySelector('.blog-title');
        const blogAuthor = blogNode.querySelector('.blog-author');
        const blogDate = blogNode.querySelector('.blog-date');
        const blogContentPreview = blogNode.querySelector('.blog-content-preview');
        const viewBtn = blogNode.querySelector('.view-btn');
        const editBtn = blogNode.querySelector('.edit-btn');
        const deleteBtn = blogNode.querySelector('.delete-btn');

        // 设置数据
        blogItem.dataset.id = blog.id;
        blogItem.dataset.userId = blog.user_id;
        blogTitle.textContent = blog.title;
        if (blog.is_private) {
            const privateBadge = document.createElement('span');
            privateBadge.className = 'private-badge';
            privateBadge.textContent = '私密';
            blogTitle.appendChild(privateBadge);
        }
        blogAuthor.textContent = blog.username || `用户 ${blog.user_id}`;
        blogDate.textContent = new Date(blog.created_at).toLocaleString();
        blogContentPreview.textContent = blog.content;
        
        // 如果是当前用户的博客，显示编辑和删除按钮
        if (currentUser && blog.user_id === currentUser.id) {
            editBtn.style.display = 'inline-block';
            deleteBtn.style.display = 'inline-block';
        }

        // 添加事件监听
        viewBtn.addEventListener('click', () => {
            window.location.href = `/blogs/${blog.id}`;
        });
        
        editBtn.addEventListener('click', () => {
            window.location.href = `/blogs/edit/${blog.id}`;
        });

        deleteBtn.addEventListener('click', () => {
            if (confirm('确定要删除这篇博客吗？')) {
                deleteBlog(blog.id, blogItem);
            }
        });

        // 添加到列表
        blogList.appendChild(blogNode);
    }
});