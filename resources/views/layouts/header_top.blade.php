<header class="navbar navbar-expand-md navbar-dark d-none d-lg-block">
    <div class="container-xl">
        <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbar-menu">
            <span class="navbar-toggler-icon"></span>
        </button>
        <h1 class="navbar-brand navbar-brand-autodark d-none-navbar-horizontal pe-0 pe-md-3">
            @foreach(\App\CodeFec\Header\functions::left() as $key => $value)
                @include($value['view'])
            @endforeach
        </h1>
        <div class="navbar-nav flex-row order-md-last">
            @foreach(\App\CodeFec\Header\functions::right() as $key => $value)
                @include($value['view'])
            @endforeach
            <div id="vue-header" class="nav-item dropdown">
                <a href="#" class="nav-link d-flex lh-1 text-reset p-0" data-bs-toggle="dropdown"
                    aria-label="Open user menu">
                    <span class="avatar avatar-sm" :style="{ 'background-image':'url('+avatar+')' }"></span>
                    <div class="d-none d-xl-block ps-2">
                        <div>@{{ Username }}</div>
                        <div class="mt-1 small text-muted">@{{ Email }}</div>
                    </div>
                </a>
                <div class="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
                    <a :href="setting_im" class="dropdown-item">个人设置</a>
                    <a href="#" @@click="logout" class="dropdown-item">Logout</a>
                </div>
            </div>
        </div>
    </div>
</header>