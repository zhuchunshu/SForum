<div class="row row-cards justify-content-center">
    <div class="col-md-12">
        <div class="border-0 card card-body topic">

            <div class="row">
                <div class="col-md-12" id="title">
                    <h1>{{$data->title}}</h1>
                </div>
                <div class="col-md-12" id="author">
                    <div class="row">
                        <div class="col-auto">
                            <span class="avatar" style="background-image: url({{super_avatar($data->user)}})"></span>
                        </div>
                        <div class="col">
                            <div class="topic-author-name">{{$data->user->username}}</div>
                            <div>发表于:{{format_date($data->created_at)}}</div>
                        </div>
                    </div>
                </div>

                <div class="col-md-12 vditor-reset" id="topic-content">
                    {!! ShortCodeR()->handle($data->content) !!}
                </div>

            </div>

        </div>
    </div>

</div>