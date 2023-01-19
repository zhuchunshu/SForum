<link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
<div class="row row-cards">

    @php($users = \App\Plugins\User\src\Models\UserFans::query()->where(['user_id' => $user->id])->paginate(15))
    <div class="col-md-12">
        <div class="border-0 card card-body">
            <h3 class="card-title">{{$user->username}} 的粉丝</h3>
            @if($users->count())
                <div class="row row-cards">
                    @foreach($users as $value)
                        <div class="col-md-6 col-lg-4">
                            <div class="card">
                                <div class="card-body p-4 text-center">
                                    <span class="avatar avatar-xl mb-3 avatar-rounded" style="background-image: url({{super_avatar($value->fans)}})"></span>
                                    <h3 class="m-0 mb-1"><a href="#">{!! u_username($value->fans) !!}</a></h3>
                                    {{--                                    <div class="text-muted">UI Designer</div>--}}
                                    <div class="mt-3">
                                        <a href="/users/group/{{$value->fans->class_id}}.html" class="badge badge-outline"
                                           style="color: {{$value->fans->Class->color}}">{{$value->fans->Class->name}}</a>
                                    </div>
                                </div>
                                <div class="d-flex">
                                    <a class="card-btn" user-click="user_follow"
                                       user-id="{{ $value->fans->id }}">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-plus"
                                             width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                             fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <circle cx="9" cy="7" r="4"></circle>
                                            <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                                            <path d="M16 11h6m-3 -3v6"></path>
                                        </svg>
                                        <span>关注</span>
                                    </a>
                                    @if((int)$value->fans->id!==auth()->id())
                                        <a href="/users/pm/{{$value->fans->id}}" class="card-btn"><!-- Download SVG icon from http://tabler-icons.io/i/phone -->
                                            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-send"
                                                 width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                                 fill="none" stroke-linecap="round" stroke-linejoin="round">
                                                <desc>Download more icon variants from https://tabler-icons.io/i/send</desc>
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                <line x1="10" y1="14" x2="21" y2="3"></line>
                                                <path d="M21 3l-6.5 18a0.55 .55 0 0 1 -1 0l-3.5 -7l-7 -3.5a0.55 .55 0 0 1 0 -1l18 -6.5"></path>
                                            </svg>
                                            私信</a>
                                    @endif

                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
                <div class="mt-2">
                    {!! make_page($users) !!}
                </div>
            @else
                <div class="empty">
                    <div class="empty-header">404</div>
                    <p class="empty-title">暂无更多结果</p>
                </div>
            @endif
        </div>
    </div>
</div>