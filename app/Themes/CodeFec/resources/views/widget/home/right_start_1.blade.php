@if(auth()->check())
    <div class="col-md-12">
        <div class="card">
            <div class="card-cover card-cover-blurred text-center" style="background-image: url({{get_user_settings(auth()->id(),'backgroundImg','/plugins/Core/image/user_background.jpg')}})">
                <span class="avatar avatar-xl avatar-thumb avatar-rounded" style="background-image: url({{super_avatar(auth()->data())}})"></span>
            </div>
            <div class="card-body text-center">
                <div class="card-title mb-1">{!! u_username(auth()->data(),['extends' => true,'home_right' => true,'link' => false]) !!}</div>
                <div class="text-muted">至今共发布{{auth()->data()->topic->count()}}篇文章</div>

            </div>
            <div class="d-flex">
                <a href="/topic/create" class="card-btn">
                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-pencil-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M8 20l10.5 -10.5a2.828 2.828 0 1 0 -4 -4l-10.5 10.5v4h4z"></path>
                        <path d="M13.5 6.5l4 4"></path>
                        <path d="M16 18h4m-2 -2v4"></path>
                    </svg>
                    发帖
                </a>
                <a href="/users/{{auth()->id()}}.html" class="card-btn">
                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <circle cx="9" cy="7" r="4"></circle>
                        <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                        <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                        <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
                    </svg>个人中心
                </a>
            </div>
        </div>
    </div>
@else
    <div class="col-md-12">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    {{get_options("web_name")}}
                </h3>
            </div>
            <div class="card-body ">
                {{get_options("description",__("app.no description"))}}
            </div>
            <div class="d-flex">
                <a href="/register" class="card-btn">{{__("app.register")}}</a>
                <a href="/login" class="card-btn">{{__("app.login")}}</a>
            </div>
        </div>
    </div>
@endif
