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
            window.location.href = `/blogs/${blogId}`;
        });
    }
    
    // 提交按钮事件监听
    if (submitBtn) {
        submitBtn.addEventListener('click', () => {
            updateBlog(blogId);
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
            
            // 如果不是当前用户的博客，无权编辑，返回博客列表页
            if (currentUser && blog.user_id !== currentUser.id) {
                window.location.href = '/blogs';
                return;
            }
            
            // 设置表单数据
            blogTitleInput.value = blog.title;
            blogContentInput.value = blog.content;
            isPrivateCheckbox.checked = blog.is_private;
        } catch (error) {
            console.error('加载博客失败:', error);
            window.location.href = '/blogs';
        }
    }
    
    // 更新博客
    async function updateBlog(id) {
        const title = blogTitleInput.value.trim();
        const content = blogContentInput.value.trim();
        const isPrivate = isPrivateCheckbox.checked;
        
        if (!title || !content) {
            alert('标题和内容不能为空！');
            return;
        }
        
        try {
            const response = await fetch(`/api/blogs/${id}`, {
                method: 'PUT',
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
                // 更新成功后跳转到博客详情页
                window.location.href = `/blogs/${id}`;
            } else {
                alert('更新博客失败！');
            }
        } catch (error) {
            console.error('更新博客失败:', error);
            alert('更新博客失败！');
        }
    }
});