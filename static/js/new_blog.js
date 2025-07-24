document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const blogTitleInput = document.getElementById('blog-title');
    const blogContentInput = document.getElementById('blog-content');
    const isPrivateCheckbox = document.getElementById('is-private');
    const submitBtn = document.getElementById('submit-btn');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    const backBtn = document.getElementById('back-btn');

    // 获取当前用户信息
    getCurrentUser();
    
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
    
    // 提交按钮事件监听
    if (submitBtn) {
        submitBtn.addEventListener('click', createBlog);
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
    
    // 创建博客
    async function createBlog() {
        const title = blogTitleInput.value.trim();
        const content = blogContentInput.value.trim();
        const isPrivate = isPrivateCheckbox.checked;
        
        if (!title || !content) {
            alert('标题和内容不能为空！');
            return;
        }
        
        try {
            const response = await fetch('/api/blogs', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    title,
                    content,
                    is_private: isPrivate
                })
            });
            
            if (response.ok) {
                const blog = await response.json();
                // 创建成功后跳转到博客详情页
                window.location.href = `/blogs/${blog.id}`;
            } else {
                alert('创建博客失败！');
            }
        } catch (error) {
            console.error('创建博客失败:', error);
            alert('创建博客失败！');
        }
    }
});