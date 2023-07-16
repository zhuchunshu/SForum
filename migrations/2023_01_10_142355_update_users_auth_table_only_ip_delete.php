<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateUsersAuthTableOnlyIpDelete extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('users_auth', function (Blueprint $table) {
            $table->text('user_agent')->nullable();
            $table->dropColumn('online');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_auth', function (Blueprint $table) {
            //
        });
    }
}
