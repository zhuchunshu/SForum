<div class="row row-cards">

    @if($user->options->qq && $user->options->wx && $user->options->website && $user->options->email)
        <div class="col-12">
            <div class="border-0 card card-body">
                <dl class="row">
                    @if($user->options->qq)
                        <dt class="col-sm-3">{{ __('user.QQ') }}:</dt>
                        <dd class="col-sm-9">{{ $user->options->qq }}</dd>
                    @endif
                    @if($user->options->wx)
                        <dt class="col-sm-3">{{ __('user.wechat') }}:</dt>
                        <dd class="col-sm-9">{{ $user->options->wx }}</dd>
                    @endif
                    @if($user->options->website)
                        <dt class="col-sm-3">{{ __('user.website') }}:</dt>
                        <dd class="col-sm-9"><a href="{{ $user->options->website }}">{{ $user->options->website }}</a>
                        </dd>
                    @endif
                    @if($user->options->email)
                        <dt class="col-sm-3">{{ __('user.email') }}:</dt>
                        <dd class="col-sm-9"><a
                                    href="mailto:{{ $user->options->email }}">{{ $user->options->email }}</a>
                        </dd>
                    @endif
                </dl>
            </div>
        </div>
    @endif

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
                            <span class="bg-orange-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
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
                            <span class="bg-yellow-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
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
                            <span class="bg-azure-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
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

                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-green-lt text-white avatar" title="{{$user->created_at}}"
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


                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-pink-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg"
                                   class="icon icon-tabler icon-tabler-message-circle" width="24" height="24"
                                   viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                   stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/message-circle</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
   <line x1="12" y1="12" x2="12" y2="12.01"></line>
   <line x1="8" y1="12" x2="8" y2="12.01"></line>
   <line x1="16" y1="12" x2="16" y2="12.01"></line>
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
                    </div>
                </div>


                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-primary-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-note"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/note</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <line x1="13" y1="20" x2="20" y2="13"></line>
   <path d="M13 20v-6a1 1 0 0 1 1 -1h6v-7a2 2 0 0 0 -2 -2h-12a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7"></path>
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
                    </div>
                </div>

                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-red-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
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

                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-indigo-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                              <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-circle"
                                   width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                   fill="none" stroke-linecap="round" stroke-linejoin="round">
   <desc>Download more icon variants from https://tabler-icons.io/i/user-circle</desc>
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="12" r="9"></circle>
   <circle cx="12" cy="10" r="3"></circle>
   <path d="M6.168 18.849a4 4 0 0 1 3.832 -2.849h4a4 4 0 0 1 3.834 2.855"></path>
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
                    </div>
                </div>

                <div class="col-12 col-md-6 col-lg-4">
                    <div class="card card-sm">
                        <div class="card-body">
                            <div class="row align-items-center">
                                <div class="col-auto">
                            <span class="bg-cyan-lt text-white avatar"><!-- Download SVG icon from http://tabler-icons.io/i/heart -->
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