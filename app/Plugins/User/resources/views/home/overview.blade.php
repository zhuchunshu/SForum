<div class="row row-cards">

    <div class="col-12">
        <div class="border-0 card card-body">
            <h3 class="card-title">个人信息</h3>
            <div class="row row-cards">

                {{--                用户组--}}
                <div class="col-12 col-md-6 col-lg-3">
                    <a href="/users/group/{{$user->class_id}}.html" class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                             <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users"
                                  width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                  fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"></path>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        <div style="display:inline-block">
                                            {{$user->Class->name}}
                                        </div>
                                    </div>
                                    <div class="text-muted">
                                        用户组
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </a>
                </div>
                {{--                qq--}}
                @if($user->options->qq)
                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm" data-bs-toggle="popover" data-bs-placement="top" title="QQ"
                             data-bs-content="{{ (string)$user->options->qq }}">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-brand-qq"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M14 7h.01"></path>
   <path d="M10 7h.01"></path>
   <path d="M6 11c4 4 8 4 12 0"></path>
   <path d="M9 13.5v2.5"></path>
   <path d="M10.5 10c.667 1.333 2.333 1.333 3 0h-3z"></path>
   <path d="M16 21c1.5 0 3.065 -1 1 -3c4.442 2 1.987 -4.5 1 -7c0 -4 -1.558 -8 -6 -8s-6 4 -6 8c-.987 2.5 -3.442 9 1 7c-2.065 2 -.5 3 1 3h8z"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div class="text-truncate text-truncate-end">
                                                {{ $user->options->qq }}
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            QQ
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @endif

                {{--                微信--}}
                @if($user->options->wx)
                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm" data-bs-toggle="popover" data-bs-placement="top"
                             title="{{ __('user.wechat') }}" data-bs-content="{{ (string)$user->options->wx }}">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg"
                                     class="icon icon-tabler icon-tabler-brand-wechat" width="24" height="24"
                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                     stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16.5 10c3.038 0 5.5 2.015 5.5 4.5c0 1.397 -.778 2.645 -2 3.47l0 2.03l-1.964 -1.178a6.649 6.649 0 0 1 -1.536 .178c-3.038 0 -5.5 -2.015 -5.5 -4.5s2.462 -4.5 5.5 -4.5z"></path>
   <path d="M11.197 15.698c-.69 .196 -1.43 .302 -2.197 .302a8.008 8.008 0 0 1 -2.612 -.432l-2.388 1.432v-2.801c-1.237 -1.082 -2 -2.564 -2 -4.199c0 -3.314 3.134 -6 7 -6c3.782 0 6.863 2.57 7 5.785l0 .233"></path>
   <path d="M10 8h.01"></path>
   <path d="M7 8h.01"></path>
   <path d="M15 14h.01"></path>
   <path d="M18 14h.01"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div class="text-truncate text-truncate-end">
                                                {{ $user->options->wx }}
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            {{ __('user.wechat') }}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @endif

                {{--                网站--}}
                @if($user->options->website)
                    <div class="col-12 col-md-6 col-lg-3">
                        <a href="{{ $user->options->website }}" class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-world-www"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M19.5 7a9 9 0 0 0 -7.5 -4a8.991 8.991 0 0 0 -7.484 4"></path>
   <path d="M11.5 3a16.989 16.989 0 0 0 -1.826 4"></path>
   <path d="M12.5 3a16.989 16.989 0 0 1 1.828 4"></path>
   <path d="M19.5 17a9 9 0 0 1 -7.5 4a8.991 8.991 0 0 1 -7.484 -4"></path>
   <path d="M11.5 21a16.989 16.989 0 0 1 -1.826 -4"></path>
   <path d="M12.5 21a16.989 16.989 0 0 0 1.828 -4"></path>
   <path d="M2 10l1 4l1.5 -4l1.5 4l1 -4"></path>
   <path d="M17 10l1 4l1.5 -4l1.5 4l1 -4"></path>
   <path d="M9.5 10l1 4l1.5 -4l1.5 4l1 -4"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div class="text-truncate text-truncate-end">
                                                {{ $user->options->website }}
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            {{ __('user.website') }}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </a>
                    </div>
                @endif


                {{--                邮箱--}}
                @if($user->options->email)
                    <div class="col-12 col-md-6 col-lg-3">
                        <a href="mailto:{{ $user->options->email }}" class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-mail"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 5m0 2a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2z"></path>
   <path d="M3 7l9 6l9 -6"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div class="text-truncate text-truncate-end">
                                                {{ $user->options->email }}
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            {{ __('user.email') }}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </a>
                    </div>
                @endif

                {{--                邮箱--}}
                @if($user->moderator->count())
                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg"
                                     class="icon icon-tabler icon-tabler-award-filled" width="24" height="24"
                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                     stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M19.496 13.983l1.966 3.406a1.001 1.001 0 0 1 -.705 1.488l-.113 .011l-.112 -.001l-2.933 -.19l-1.303 2.636a1.001 1.001 0 0 1 -1.608 .26l-.082 -.094l-.072 -.11l-1.968 -3.407a8.994 8.994 0 0 0 6.93 -3.999z"
         stroke-width="0" fill="currentColor"></path>
   <path d="M11.43 17.982l-1.966 3.408a1.001 1.001 0 0 1 -1.622 .157l-.076 -.1l-.064 -.114l-1.304 -2.635l-2.931 .19a1.001 1.001 0 0 1 -1.022 -1.29l.04 -.107l.05 -.1l1.968 -3.409a8.994 8.994 0 0 0 6.927 4.001z"
         stroke-width="0" fill="currentColor"></path>
   <path d="M12 2l.24 .004a7 7 0 0 1 6.76 6.996l-.003 .193l-.007 .192l-.018 .245l-.026 .242l-.024 .178a6.985 6.985 0 0 1 -.317 1.268l-.116 .308l-.153 .348a7.001 7.001 0 0 1 -12.688 -.028l-.13 -.297l-.052 -.133l-.08 -.217l-.095 -.294a6.96 6.96 0 0 1 -.093 -.344l-.06 -.271l-.049 -.271l-.02 -.139l-.039 -.323l-.024 -.365l-.006 -.292a7 7 0 0 1 6.76 -6.996l.24 -.004z"
         stroke-width="0" fill="currentColor"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div class="text-truncate text-truncate-end">
                                                <div class="avatar-list avatar-list-stacked" data-bs-toggle="modal"
                                                     data-bs-target="#modal-user-moderator">
                                                    @foreach($user->moderator as $moderator)
                                                        <span class="avatar avatar-rounded bg-azure-lt"
                                                              style="--tblr-avatar-size:25px;">
                                                        {!! $moderator->tag->icon !!}
                                                    </span>
                                                    @endforeach
                                                </div>
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            管理板块
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @endif

            </div>
        </div>
    </div>

    <div class="col-12">
        <div class="border-0 card card-body">
            <h3 class="card-title">{{__("user.data")}}</h3>
            <div class="row row-cards">
                @if(get_options("user_location_show_close",'false')!=="true")
                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-location"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M21 3l-6.5 18a0.55 .55 0 0 1 -1 0l-3.5 -7l-7 -3.5a0.55 .55 0 0 1 0 -1l18 -6.5"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div id="vue-users-data-session-ip" style="display:inline-block">
                                            <span v-if="ip">
                                                @{{ ip }}
                                            </span>
                                                <span v-else>
                                                Loading<span class="animated-dots"></span>
                                            </span>
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            {{__("user.location")}}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @else
                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-id"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 4m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v10a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z"></path>
   <path d="M9 10m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0"></path>
   <path d="M15 8l2 0"></path>
   <path d="M15 12l2 0"></path>
   <path d="M7 16l10 0"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <div id="vue-users-data-session-ip" style="display:inline-block">
                                                {{$user->id}}
                                            </div>
                                        </div>
                                        <div class="text-muted">
                                            UID
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                @endif
                <div class="col-12 col-md-6 col-lg-3">
                    <a href="/users/{{$user->id}}.html?m=users_home_menu_2" class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-book"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 19a9 9 0 0 1 9 0a9 9 0 0 1 9 0"></path>
   <path d="M3 6a9 9 0 0 1 9 0a9 9 0 0 1 9 0"></path>
   <line x1="3" y1="6" x2="3" y2="19"></line>
   <line x1="12" y1="6" x2="12" y2="19"></line>
   <line x1="21" y1="6" x2="21" y2="19"></line>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        {{$user->topic->count()}}
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.topic count")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                <div class="col-12 col-md-6 col-lg-3">
                    <a href="/users/{{$user->id}}.html?m=users_home_menu_3" class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 21v-13a3 3 0 0 1 3 -3h10a3 3 0 0 1 3 3v6a3 3 0 0 1 -3 3h-9l-4 4"></path>
   <line x1="8" y1="9" x2="16" y2="9"></line>
   <line x1="8" y1="13" x2="14" y2="13"></line>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        {{$user->comments->count()}}
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.comment count")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                <div class="col-12 col-md-6 col-lg-3">
                    <a href="/users/{{$user->id}}.html?m=users_home_menu_5" class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        {{$user->fan->count()}}
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.fans count")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </a>
                </div>
            </div>
        </div>
    </div>

    @if(auth()->id() ===(int)$user->id)
        <div class="col-12">

            <div class="border-0 card card-body">
                <h3 class="card-title">{{__("user.wealth")}}</h3>
                <div class="row row-cards">

                    <div class="col-12 col-md-6 col-lg-3">
                        <a href="/user/asset/money" class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-cyan-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg"
                                   class="icon icon-tabler icon-tabler-currency-dollar" width="24" height="24"
                                   viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                   stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/currency-dollar</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M16.7 8a3 3 0 0 0 -2.7 -2h-4a3 3 0 0 0 0 6h4a3 3 0 0 1 0 6h-4a3 3 0 0 1 -2.7 -2"></path>
   <path d="M12 3v3m0 12v3"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <span>{{$user->options->money}}</span>
                                        </div>
                                        <div class="text-muted">
                                            {{get_options('wealth_money_name','余额')}}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </a>
                    </div>

                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-cyan-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-credit-card"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/credit-card</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <rect x="3" y="5" width="18" height="14" rx="3"></rect>
   <line x1="3" y1="10" x2="21" y2="10"></line>
   <line x1="7" y1="15" x2="7.01" y2="15"></line>
   <line x1="11" y1="15" x2="13" y2="15"></line>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <span>{{$user->options->credits}}</span>
                                        </div>
                                        <div class="text-muted">
                                            {{get_options('wealth_credit_name','积分')}}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-cyan-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-coin"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/coin</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="12" r="9"></circle>
   <path d="M14.8 9a2 2 0 0 0 -1.8 -1h-2a2 2 0 0 0 0 4h2a2 2 0 0 1 0 4h-2a2 2 0 0 1 -1.8 -1"></path>
   <path d="M12 6v2m0 8v2"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <span>{{$user->options->golds}}</span>
                                        </div>
                                        <div class="text-muted">
                                            {{get_options('wealth_golds_name','金币')}}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>


                    <div class="col-12 col-md-6 col-lg-3">
                        <div class="card card-sm">
                            <div class="card-body">
                                <div class="row align-items-center">
                                    <div class="col-auto">
                            <span class="bg-cyan-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-activity"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/activity</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 12h4l3 8l4 -16l3 8h4"></path>
</svg>
                            </span>
                                    </div>
                                    <div class="col">
                                        <div class="font-weight-medium">
                                            <span>{{$user->options->exp}}</span>
                                        </div>
                                        <div class="text-muted">
                                            {{get_options('wealth_exp_name','经验')}}
                                        </div>
                                    </div>
                                    <div class="col-auto">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>


                </div>
            </div>

        </div>
    @endif


    <div class="col-12">
        <div class="border-0 card card-body">
            <div class="row row-cards">

                <div class="col-12 col-md-6 col-lg-3">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-indigo-lt text-white avatar" title="{{$user->created_at}}"
                                  data-bs-toggle="tooltip" data-bs-placement="top"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-clock"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="12" r="9"></circle>
   <polyline points="12 7 12 12 15 15"></polyline>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        <span title="{{$user->created_at}}" data-bs-toggle="tooltip"
                                              data-bs-placement="top">{{format_date($user->created_at)}}</span>
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.register time")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-12 col-md-6 col-lg-3">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-indigo-lt text-white avatar" title="{{$user->created_at}}"
                                  data-bs-toggle="tooltip" data-bs-placement="top"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-clock"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="12" r="9"></circle>
   <polyline points="12 7 12 12 15 15"></polyline>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        @if(!$user->auth->count())
                                            <span title="查询失败" data-bs-toggle="tooltip"
                                                  data-bs-placement="top">查询失败</span>
                                        @else
                                            <span title="{{$user->auth->last()->created_at}}" data-bs-toggle="tooltip"
                                                  data-bs-placement="top">{{format_date($user->auth->last()->created_at)}}</span>
                                        @endif
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.last login time")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </div>
                </div>


                <div class="col-12 col-md-6 col-lg-3">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-indigo-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-star"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/star</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        {{$user->collections->count()}}
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.collection count")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </div>
                </div>


                <div class="col-12 col-md-6 col-lg-3">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-indigo-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-tag"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/tag</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="8.5" cy="8.5" r="1" fill="currentColor"></circle>
   <path d="M4 7v3.859c0 .537 .213 1.052 .593 1.432l8.116 8.116a2.025 2.025 0 0 0 2.864 0l4.834 -4.834a2.025 2.025 0 0 0 0 -2.864l-8.117 -8.116a2.025 2.025 0 0 0 -1.431 -.593h-3.859a3 3 0 0 0 -3 3z"></path>
</svg>
                            </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">
                                        {{$user->tags->count()}}
                                    </div>
                                    <div class="text-muted">
                                        {{__("user.topic tag count")}}
                                    </div>
                                </div>
                                <div class="col-auto">
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
    </div>
</div>