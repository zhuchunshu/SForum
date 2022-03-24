<span class="hr-text text-red">隐藏内容</span>
<div class="card-title text-center">
    隐藏内容，回复后可见
    @if(!auth()->check())
        <p>
            <a href="/login" class="btn btn-square btn-dark">登陆</a>
            <a href="/register" class="btn btn-square btn-light">注册</a>
        </p>
    @endif
</div>
<span class="hr-text text-red">end</span>