<div class="col-md-12">
    <div class="border-0 card">
        <div class="card-header">
            <div class="card-title">评论</div>
        </div>
        <div class="card-body">
            @if(!auth()->check())
                <div class="col-md-12">
                    <div class="border-0 card">
                        <div class="empty">
                            <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                    <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"></path>
                                    <path d="M20 12h-13l3 -3m0 6l-3 -3"></path>
                                </svg>
                            </div>
                            <p class="empty-title">无权限</p>
                            <p class="empty-subtitle text-muted">
                                请登录后评论
                            </p>
                            @if(!auth()->check())
                                <div class="empty-action">
                                    <a href="/login" class="btn btn-primary">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/search -->
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"></path>
                                            <path d="M20 12h-13l3 -3m0 6l-3 -3"></path>
                                        </svg>
                                        登陆
                                    </a>
                                </div>
                            @endif
                        </div>
                    </div>
                </div>
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

                <div class="mb-1">
                    <form action="/topic/create/comment/{{$data->id}}" method="post">
                        <div class="row">
                            <div class="col-md-12 mb-3">
                                <x-csrf/>
                                <input type="hidden" name="topic_id" value="{{$data->id}}">
                                <textarea name="content" placeholder="说点什么..." class="form-control" data-bs-toggle="autosize" required>{{request()->input('content')}}</textarea>
                            </div>
                            <div class="col-12">
                                <div class="row">
                                    <div class="col">
                                        <button class="btn btn-primary">评论</button>
                                    </div>
                                    <div class="col-auto"><a class="text-muted" href="/topic/create/comment/{{$data->id}}">[高级回复]</a></div>
                                </div>
                            </div>
                        </div>
                    </form>
                </div>
            @endif
        </div>
    </div>
</div>