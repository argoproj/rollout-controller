import {faQuestionCircle} from '@fortawesome/free-regular-svg-icons';
import {faCheck, faCircleNotch, faClipboard, faExclamationTriangle, faTimes} from '@fortawesome/free-solid-svg-icons';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import * as React from 'react';
import {GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1ReplicaSetInfo} from '../../../models/rollout/generated';
import {Pod} from '../../../models/rollout/rollout';
import {Menu} from '../menu/menu';
import {ReplicaSetStatus, ReplicaSetStatusIcon} from '../status-icon/status-icon';
import {ThemeDiv} from '../theme-div/theme-div';
import {Tooltip} from '../tooltip/tooltip';
import {WaitFor} from '../wait-for/wait-for';

import './pods.scss';

export enum PodStatus {
    Pending = 'pending',
    Success = 'success',
    Failed = 'failure',
    Warning = 'warning',
    Unknown = 'unknown',
}

export const ParsePodStatus = (status: string): PodStatus => {
    switch (status) {
        case 'Pending':
        case 'Terminating':
        case 'ContainerCreating':
            return PodStatus.Pending;
        case 'Running':
        case 'Completed':
            return PodStatus.Success;
        case 'Failed':
        case 'InvalidImageName':
        case 'CrashLoopBackOff':
            return PodStatus.Failed;
        case 'ImagePullBackOff':
        case 'RegistryUnavailable':
            return PodStatus.Warning;
        default:
            return PodStatus.Unknown;
    }
};

export const PodIcon = (props: {status: string}) => {
    const {status} = props;
    let icon;
    let spin = false;
    if (status.startsWith('Init:')) {
        icon = faCircleNotch;
        spin = true;
    }
    if (status.startsWith('Signal:') || status.startsWith('ExitCode:')) {
        icon = faTimes;
    }
    if (status.endsWith('Error') || status.startsWith('Err')) {
        icon = faExclamationTriangle;
    }

    const className = ParsePodStatus(status);

    switch (className) {
        case PodStatus.Pending:
            icon = faCircleNotch;
            spin = true;
            break;
        case PodStatus.Success:
            icon = faCheck;
            break;
        case PodStatus.Failed:
            icon = faTimes;
            break;
        case PodStatus.Warning:
            icon = faExclamationTriangle;
            break;
        default:
            spin = false;
            icon = faQuestionCircle;
            break;
    }

    return (
        <ThemeDiv className={`pod-icon pod-icon--${className}`}>
            <FontAwesomeIcon icon={icon} spin={spin} />
        </ThemeDiv>
    );
};

export const ReplicaSet = (props: {rs: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1ReplicaSetInfo}) => {
    const rsName = props.rs.objectMeta.name;
    return (
        <ThemeDiv className='pods'>
            {rsName && (
                <ThemeDiv className='pods__header'>
                    <span style={{marginRight: '5px'}}>{rsName}</span> <ReplicaSetStatusIcon status={props.rs.status as ReplicaSetStatus} />
                </ThemeDiv>
            )}
            {props.rs.pods && props.rs.pods.length > 0 && (
                <ThemeDiv className='pods__container'>
                    <WaitFor loading={(props.rs.pods || []).length < 1}>
                        {props.rs.pods.map((pod, i) => (
                            <PodWidget key={pod.objectMeta.uid} pod={pod} />
                        ))}
                    </WaitFor>
                </ThemeDiv>
            )}
        </ThemeDiv>
    );
};

export const PodWidget = (props: {pod: Pod}) => (
    <Menu items={[{label: 'Copy Name', action: () => navigator.clipboard.writeText(props.pod.objectMeta?.name), icon: faClipboard}]}>
        <Tooltip
            content={
                <div>
                    <div>Status: {props.pod.status}</div>
                    <div>{props.pod.objectMeta?.name}</div>
                </div>
            }>
            <PodIcon status={props.pod.status} />
        </Tooltip>
    </Menu>
);