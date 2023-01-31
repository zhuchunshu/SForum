<div class="col-md-12" id="author">
    <div class="row">
        <div class="col">
            <div class="row">
                <div class="col-auto">
                    <a class="avatar" href="/users/{{ $data->user->id }}.html"
                       style="background-image: url({{ super_avatar($data->user) }})"></a>
                </div>
                <div class="col" style="margin-left: -10px">
                    <div class="topic-author-name">
                        {!! u_username($data->user,['topic' => true,'extends' => true,'class' => ['text-reset']]) !!}
                        <a data-bs-toggle="tooltip" data-bs-placement="right" title="{{$data->user->class->name}}" href="/users/group/{{$data->user->class->id}}.html" style="color:{{$data->user->class->color}}">
                            <span>{!! $data->user->class->icon !!}</span>
                        </a>
                    </div>
                    <div>{{__("app.Published on")}}:{{ format_date($data->created_at) }}
                        <span v-if="user.city">
                            |
                            <span class="text-red">
                                @{{user.city}}
                            </span>
                        </span>
                    </div>
                </div>
            </div>
        </div>
        <div class="col-auto">

        </div>
    </div>
</div>