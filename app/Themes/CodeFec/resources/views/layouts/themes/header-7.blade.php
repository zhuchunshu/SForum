<div class="wrapper sticky-top border-top-wide border-primary">
    <div class="navbar-expand-md">
        <div class="collapse navbar-collapse" id="navbar-menu">
            <div class="navbar navbar-dark">
                <div class="container-xl">
                    @include('App::layouts.themes.menu')
                    @include('App::layouts.themes.right')
                </div>
            </div>
        </div>
    </div>
    <header class="navbar navbar-expand-md navbar-dark d-print-none">
        <div class="container-xl">
            <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbar-menu">
                <span class="navbar-toggler-icon"></span>
            </button>
            <h1 class="navbar-brand navbar-brand-autodark d-none-navbar-horizontal pe-0 pe-md-3">
                <a href="/"> @if(!get_options('web_logo')){{get_options('web_name', 'CodeFec')}}@else
                        <img src="{{get_options('web_name', 'CodeFec')}}" alt="{{ get_options('web_name', 'CodeFec') }}"> @endif</a>
            </h1>
            <div class="flex-row navbar-nav order-md-last">
                @include('App::layouts.themes.header-right')
            </div>
        </div>
    </header>
</div>
