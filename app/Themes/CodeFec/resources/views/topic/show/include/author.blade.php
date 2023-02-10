<div class="mx-3 my-3 mb-0" id="author">
    <div class="row">
        <div class="col">
            <div class="row">
                <div class="col-auto">
                    <a class="avatar avatar-rounded" href="/users/{{ $data->user->id }}.html"
                       style="background-image: url({{ super_avatar($data->user) }})"></a>
                </div>
                <div class="col" style="margin-left: -10px">
                    <div class="topic-author-name">
                        {!! u_username($data->user,['topic' => true,'extends' => true,'class' => ['text-reset']]) !!}
                    </div>
                    <div>
                        <span class="cursor-pointer" data-bs-toggle="tooltip" data-bs-placement="bottom" title="{{$data->created_at}}">
                            {{__("app.Published on")}}:{{ format_date($data->created_at) }}
                        </span>
                        ｜<span class="cursor-pointer" data-bs-toggle="tooltip" data-bs-placement="bottom" title="{{$data->updated_at}}">{{__("app.Updated on")}}:{{ format_date($data->created_at) }}</span>
                        <span v-if="user.city">
                            |
                            <span class="text-red">
                                @{{user.city}}
                            </span>
                        </span>
                    </div>
                </div>
                <div class="col-auto">

                </div>
            </div>
        </div>
        <div class="col-auto">

        </div>
    </div>
</div>
<div class="mt-0 mb-0">
    <div class="hr-text hr-text-right mt-3 mb-0">
        <div class="text-muted">
{{--            浏览量--}}
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-eye" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <path d="M12 12m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0"></path>
                <path d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7"></path>
            </svg>
            {{$data->view}}
        </div>
        <span class="mx-1">|</span>
        <a class="text-muted" href="#comment"><svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message-circle" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
                <path d="M12 12l0 .01"></path>
                <path d="M8 12l0 .01"></path>
                <path d="M16 12l0 .01"></path>
            </svg>
            {{$comment->total()}}
        </a>
    </div>
</div>