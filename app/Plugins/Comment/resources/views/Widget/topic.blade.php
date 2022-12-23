<div class="col-md-12">
    <div class="border-0 card">
        <div class="card-header">
            <div class="card-title">评论</div>
        </div>
        <div class="card-body">
            @if(!auth()->check())
                <a href="/login" class="btn btn-dark">登陆</a>
                OR
                <a href="/register" class="btn btn-light">注册</a>
            @else
                <div class="alert alert-important alert-info alert-dismissible" role="alert">
                    <div class="d-flex">
                        <div>
                            <!-- Download SVG icon from http://tabler-icons.io/i/info-circle -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><circle cx="12" cy="12" r="9"></circle><line x1="12" y1="8" x2="12.01" y2="8"></line><polyline points="11 12 12 12 12 16 13 16"></polyline></svg>
                        </div>
                        <div>
                            讨论应以学习和精进为目的。请勿发布不友善或者负能量的内容，与人为善，比聪明更重要！
                        </div>
                    </div>
                    <a class="btn-close btn-close-white" data-bs-dismiss="alert" aria-label="close"></a>
                </div>

                <div class="mb-1" id="topic-comment-model">
                    <form action="" method="post" @@submit.prevent="submit">
                        <div class="row">
                            <div class="col-md-12">
                                <div id="topic-comment"></div>
                            </div>
                        </div>
                        <button class="btn btn-azure">评论</button>
                    </form>
                </div>
            @endif
        </div>
    </div>
</div>