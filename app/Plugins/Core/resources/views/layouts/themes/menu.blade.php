<div class="rounded-lg shadow bg-base-200 drawer drawer-mobile h-52">
    <input id="my-drawer-2" type="checkbox" class="drawer-toggle">
    <div class="flex flex-col items-center justify-center drawer-content">
        <label for="my-drawer-2" class="mb-4 btn btn-primary drawer-button lg:hidden">open menu</label>
        <div class="hidden text-xs text-center lg:block">Menu is always open on desktop size.
            <br>Resize the browser to see toggle button on mobile size
        </div>
        <div class="text-xs text-center lg:hidden">Menu can be toggled on mobile size.
            <br>Resize the browser to see fixed sidebar on desktop size
        </div>
    </div>
    <div class="drawer-side">
        <label for="my-drawer-2" class="drawer-overlay"></label>
        <ul class="menu p-4 overflow-y-auto w-80 bg-base-100 text-base-content">
            <li>
                <a>Menu Item</a>
            </li>
            <li>
                <a>Menu Item</a>
            </li>
        </ul>
    </div>
</div>

<div id="common-menu" class="hidden">
    <ul class="menu py-4 shadow-lg bg-base-100 rounded-box">
        <li class="menu-title">
            <span>
                Menu Title
            </span>
        </li>
        <li class="bordered disabled">
            <a>
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    class="inline-block w-5 h-5 mr-2 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path>
                </svg>
                Item is disabled

            </a>
        </li>
        <li class="bordered">
            <a>
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    class="inline-block w-5 h-5 mr-2 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path>
                </svg>
                It has border

            </a>
        </li>
        <li class="hover-bordered">
            <a>
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    class="inline-block w-5 h-5 mr-2 stroke-current">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path>
                </svg>
                It shows border on hover

            </a>
        </li>
    </ul>
</div>
