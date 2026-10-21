import type { MessageKey } from '#lib/i18n/index.ts';
import type { CurrentEmployee } from '#lib/types.ts';

export interface NavItem {
	href: string;
	label: MessageKey;
}

export interface NavSection {
	title: MessageKey;
	items: NavItem[];
}

/** Menu by role. Visibility is a UI convenience only; the API authorizes every request. */
export function navigation(user: CurrentEmployee, isReviewer: boolean): NavSection[] {
	const sections: NavSection[] = [
		{
			title: 'nav.myLeave',
			items: [
				{ href: '/dashboard', label: 'nav.dashboard' },
				{ href: '/leave/new', label: 'nav.newRequest' },
				{ href: '/leave/history', label: 'nav.history' },
				{ href: '/notifications', label: 'nav.notifications' }
			]
		}
	];
	if (isReviewer) {
		sections.push({
			title: 'nav.team',
			items: [
				{ href: '/leader/dashboard', label: 'nav.leaderDashboard' },
				{ href: '/leader/requests', label: 'nav.leaderRequests' }
			]
		});
	}
	if (user.role === 'ADMIN') {
		sections.push({
			title: 'nav.admin',
			items: [
				{ href: '/admin/dashboard', label: 'nav.adminDashboard' },
				{ href: '/admin/employees', label: 'nav.employees' },
				{ href: '/admin/holidays', label: 'nav.holidays' },
				{ href: '/admin/settings', label: 'nav.settings' },
				{ href: '/admin/notifications', label: 'nav.outbox' },
				{ href: '/admin/email-log', label: 'nav.emailLog' },
				{ href: '/admin/system', label: 'nav.system' }
			]
		});
	}
	return sections;
}

export function isCurrent(pathname: string, href: string): boolean {
	return pathname === href || pathname.startsWith(href + '/');
}
