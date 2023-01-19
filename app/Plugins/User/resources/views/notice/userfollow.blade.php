<div class="row row-cards justify-content-center">
    <div class="col-md-12">
        <div class="p-4 text-center">
            <span class="avatar avatar-xl mb-3 avatar-rounded" style="background-image: url({{super_avatar(auth()->data())}})"></span>
            <h3 class="m-0 mb-1"><a href="/users/{{auth()->id()}}.html">{{auth()->data()->username}}</a></h3>
            <div class="text-muted">关注了你!</div>
        </div>
    </div>
    <div class="col-md-12 text-center">
        <a class="btn btn-light cursor-pointer" user-click="user_follow" user-id="{{ auth()->id() }}">
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <circle cx="9" cy="7" r="4"></circle>
                <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                <path d="M16 11h6m-3 -3v6"></path>
            </svg>
            <span>关注</span>
        </a>
    </div>
</div>