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
                            @if(get_options("comment_emoji_close",'false')!=="true" && count((new \App\Plugins\Core\src\Lib\Emoji())->get()))
                                <div class="col-md-4">
                                    <div class="card">
                                        <ul class="nav nav-tabs" data-bs-toggle="tabs" style="flex-wrap: inherit;
        width: 100%;
        height: 3.333333rem;
        padding: 0.373333rem 0.32rem 0;
        box-sizing: border-box;
        /* 下面是实现横向滚动的关键代码 */
        display: inline;
        float: left;
        white-space: nowrap;
        overflow-x: scroll;
        -webkit-overflow-scrolling: touch; /*解决在ios滑动不顺畅问题*/
        overflow-y: hidden;">
                                            @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                <li class="nav-item">
                                                    <a href="#emoji-list-{{$key}}"
                                                       class="nav-link @if ($loop->first) active @endif"
                                                       data-bs-toggle="tab">{{$key}}</a>
                                                </li>
                                            @endforeach
                                        </ul>
                                        <div class="card-body">
                                            <div class="tab-content">
                                                @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                    <div class="tab-pane  @if ($loop->first) active @endif show"
                                                         id="emoji-list-{{$key}}" style="max-height: 220px;overflow-x: hidden;">
                                                        <div class="row">
                                                            @if($value['type'] === 'image')
                                                                @foreach($value['container'] as $emojis)
                                                                    <div @@click="selectEmoji('{{$emojis['text']}}')"
                                                                         class="col-3 col-sm-2 col-md-4 col-lg-3 hvr-glow emoji-picker"
                                                                         emoji-data="{{$emojis['text']}}">{!! $emojis['icon'] !!}</div>
                                                                @endforeach
                                                            @endif
                                                        </div>
                                                    </div>
                                                @endforeach
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-8">
                                    <div id="topic-comment"></div>
                                </div>
                            @else
                                <div class="col-md-12">
                                    <div id="topic-comment"></div>
                                </div>
                            @endif
                        </div>
                        <button class="btn btn-azure">评论</button>
                    </form>
                </div>
            @endif
        </div>
    </div>
</div>