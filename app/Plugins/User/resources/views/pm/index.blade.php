@extends("App::app")

@section('title','私信 「'. $user->username.' 」')

@section('content')

    <div class="border-0 card card-body" id="user-pm-container">
        <div class="row row-cards justify-content-center">
            <div class="col-lg-3">
                <div class="border-0 card">
                    <div class="card-header">
                        <h3 class="card-title">联系人</h3>
                    </div>
                    <div class="list-group list-group-flush overflow-auto" style="max-height: 44rem">
                        @foreach($contacts as $contact)
                            <div class="list-group-item @if($contact->id===$user->id) active @endif">
                                <div class="row">
                                    <div class="col-auto">
                                        <a href="/users/pm/{{$contact->id}}">
                                            <span class="avatar"
                                                  style="background-image: url({{super_avatar($contact)}})"></span>
                                        </a>
                                    </div>
                                    <div class="col text-truncate">
                                        <a href="/users/pm/{{$contact->id}}"
                                           class="text-body d-block">{{$contact->username}}</a>
                                        <div class="text-muted text-truncate mt-n1">@if($contact->options->qianming && $contact->options->qianming!=='no bio')
                                                {{$contact->options->qianming}}
                                            @else
                                                {{__("user.no bio")}}
                                            @endif</div>
                                    </div>
                                    @if($contact->id!==$user->id && $contact->msgCount>0)
                                        <div class="col-auto">
                                            <span class="badge badge-sm badge-pill bg-red">{{$contact->msgCount}}</span>
                                        </div>
                                    @endif
                                </div>
                            </div>
                        @endforeach
                    </div>
                </div>
            </div>
            <div class="col-lg-9">
                <div class="border-0 card">
                    <div class="card-header">
                        <h3 class="card-title"><a href="/users/{{$user->id}}.html" class="avatar avatar-sm"
                                                  style="background-image: url({{super_avatar($user)}})"></a>
                            正在与: {{$user->username}} 沟通</h3>
                    </div>
                    <div class="card-body">
                        <div class="row">
                            <div id="chat-list" class="col-12 overflow-auto border-1 card card-body"
                                 style="height: 34rem">
                                @if(!$msgExists)
                                    <div class="text-center text-muted">
                                        你们至今还没有聊过
                                    </div>
                                @else

                                <div v-if="messages">
                                    <div v-for="message in messages">
                                        <div class="d-flex flex-row-reverse" v-if="message.from_id=={{auth()->id()}}">
                                            <div class="p-2 bd-highlight">
                                                    <span class="avatar avatar-sm avatar-rounded"
                                                          style="background-image: url({{super_avatar(auth()->data())}})"></span>
                                            </div>
                                            <div class="p-2">
                                                <div class="message1" v-html="message.message"></div>
                                                <span>
                                                    <small v-text="message.created_at"></small>
                                                </span>
                                            </div>
                                        </div>
                                        <div class="d-flex" v-else>
                                            <div class="p-2">
                                                    <span class="avatar avatar-sm avatar-rounded"
                                                          style="background-image: url({{super_avatar($user)}})"></span>
                                            </div>
                                            <div class="p-2">
                                                <div class="message2" v-html="message.message"></div>
                                                <span>
                                                    <small v-text="message.created_at"></small>
                                                </span>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                @endif
                            </div>
                            <div class="col-12 mt-2">
                                <div class="row">
                                    <div class="col">
                                        <div class="OwO" aria-label="Button">
                                            <svg xmlns="http://www.w3.org/2000/svg"
                                                 class="icon icon-tabler icon-tabler-mood-smile" width="24" height="24"
                                                 viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                                 stroke-linecap="round" stroke-linejoin="round">
                                                <desc>Download more icon variants from
                                                    https://tabler-icons.io/i/mood-smile
                                                </desc>
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                <circle cx="12" cy="12" r="9"></circle>
                                                <line x1="9" y1="10" x2="9.01" y2="10"></line>
                                                <line x1="15" y1="10" x2="15.01" y2="10"></line>
                                                <path d="M9.5 15a3.5 3.5 0 0 0 5 0"></path>
                                            </svg>
                                        </div>
                                        <textarea placeholder="说点啥好呢"
                                                  maxlength="{{get_options('pm_msg_maxlength',300)}}" v-model="msg"
                                                  rows="1" class="form-control OwO-textarea"></textarea>
                                    </div>
                                    <div class="col-auto d-flex align-items-end">
                                        <button @@click="sendMsg" class="btn btn-outline-tabler align-bottom"
                                                :class="{disabled:btn.disabled}">发送
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    <link rel="stylesheet" href="{{file_hash('plugins/User/css/pm.css')}}">
    <style>
        .OwO .OwO-logo {
            border: unset;
            color: unset;
            background: unset;
        }

        .message1, .message2 {
            margin: 9px auto;
            background-color: green;
            border-bottom-color: green; /*为了给after伪元素自动继承*/
            color: #fff;
            font-size: 15px;
            font-family: Arial;
            line-height: 18px;
            padding: 5px 12px 5px 12px;
            box-sizing: border-box;
            border-radius: 6px;
            position: relative;
            word-break: break-all;
        }

        .message1::after {
            content: '';
            position: absolute;
            top: 0;
            border-width: 0 0 10px 20px;
            border-style: solid;
            border-bottom-color: inherit; /*自动继承父元素的border-bottom-color*/
            border-left-color: transparent;
            border-radius: 0 0 70px 0;
        }

        /** 通过对小正方形旋转45度解决 **/
        .message2::before {
            content: '';
            position: absolute;
            top: 0;
            border-width: 0 0 10px 20px;
            border-style: solid;
            border-bottom-color: inherit; /*自动继承父元素的border-bottom-color*/
            border-left-color: transparent;
            border-radius: 0 0 5px 10px;
            left: -5px;
        }
    </style>
@endsection

@section('scripts')
    <script>var pm_socket = "{{ws_url('/User/Pm')}}?login-token={{auth()->token()}}&to={{$user->id}}";
        var msgExists = "{{$msgExists}}"
        var to_id = {{$user->id}};
        var from_user = @json(['username' => auth()->data()->username,'avatar' =>super_avatar(auth()->data())]);
        var to_user = @json(['username' => $user->username,'avatar' =>super_avatar($user)]);
    </script>
    <script src="{{file_hash('plugins/Core/js/socket.io.js')}}"></script>
    <script src="{{file_hash('plugins/User/js/pm.js')}}"></script>
@endsection