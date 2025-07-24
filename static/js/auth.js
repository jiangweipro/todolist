document.addEventListener('DOMContentLoaded', () => {
    // 获取表单元素
    const loginForm = document.querySelector('form[action="/login"]');
    const registerForm = document.querySelector('form[action="/register"]');
    
    // 登录表单处理
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const username = document.getElementById('username').value.trim();
            const password = document.getElementById('password').value.trim();
            
            if (!username || !password) {
                alert('请填写用户名和密码');
                return;
            }
            
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ username, password })
                });
                
                if (response.ok) {
                    window.location.href = '/';
                } else {
                    const data = await response.json();
                    alert(data.error || '登录失败，请检查用户名和密码');
                }
            } catch (error) {
                console.error('登录请求失败:', error);
                alert('登录请求失败，请稍后再试');
            }
        });
    }
    
    // 注册表单处理
    if (registerForm) {
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const username = document.getElementById('username').value.trim();
            const password = document.getElementById('password').value.trim();
            
            if (!username || !password) {
                alert('请填写用户名和密码');
                return;
            }
            
            if (password.length < 6) {
                alert('密码长度至少为6个字符');
                return;
            }
            
            try {
                const response = await fetch('/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ username, password })
                });
                
                if (response.ok) {
                    alert('注册成功，请登录');
                    window.location.href = '/login';
                } else {
                    const data = await response.json();
                    alert(data.error || '注册失败，该用户名可能已被使用');
                }
            } catch (error) {
                console.error('注册请求失败:', error);
                alert('注册请求失败，请稍后再试');
            }
        });
    }
});