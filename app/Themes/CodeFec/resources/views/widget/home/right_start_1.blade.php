<div class="col-md-12">
    <div class="card">
        <div class="card-status-top bg-primary"></div>
        <div class="card-body">
            <h3 class="card-title">
                {{get_options("web_name")}}
            </h3>
            <p>
                {{get_options("description",__("app.no description"))}}
            </p>
        </div>
        <div class="card-footer">
            @if(auth()->check())
                <a href="/topic/create" class="btn btn-dark">{{__("topic.create")}}</a>
            @else
                <a href="/login" class="btn btn-dark">{{__("app.login")}}</a>
                <a href="/register" class="btn btn-light">{{__("app.register")}}</a>
            @endif
        </div>
    </div>
</div>