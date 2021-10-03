<div class="shortcode-content-hidden">
    <div class="shortcode-content-hidden-info">
        <div class="row">
            <div class="col-md-12">
                <div class="row">
                    <div class="col">
                        <span class="shortcode-content-hidden-title">
                            隐藏内容，回复后可见
                        </span>
                    </div>
                    @if(!auth()->check())
                        <div class="col-auto">
                            <a href="/login" class="btn btn-square btn-dark">登陆</a>
                            <a href="/register" class="btn btn-square btn-light">注册</a>
                        </div>
                    @endif

                </div>
            </div>
            <div class="col-md-12 shortcode-content-hidden-info">
                <div class="skeleton-line"></div>
                <div class="skeleton-line"></div>
                <div class="skeleton-line"></div>
                <div class="skeleton-line"></div>
            </div>
        </div>
    </div>
</div>