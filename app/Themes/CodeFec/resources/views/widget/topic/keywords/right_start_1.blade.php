<div class="col-md-12">
    <div class="card">

        <div class="card-status-top bg-primary"></div>

        <div class="card-body">
            <h3 class="card-title">
                {{$data->name}}
            </h3>
            <p>
                标签: <b>{{$data->name}}</b> 下的帖子
            </p>
        </div>

        <div class="card-footer">
            @if(auth()->check())
                <a href="/topic/create" class="btn btn-dark">{{__("topic.create")}}</a>
            @else
                <a href="/login" class="btn btn-dark">登陆</a>
                <a href="/register" class="btn btn-light">注册</a>
            @endif
        </div>

    </div>
</div>